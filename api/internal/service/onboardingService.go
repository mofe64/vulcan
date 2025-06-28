package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mofe64/vulkan/api/internal/config"
	"github.com/mofe64/vulkan/api/internal/db/repository"
	platformv1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// note -> for attached clusters, we use the in-cluster kubeconfig to connect to the cluster.
// the in-cluster kubeconfig is the kubeconfig that is automatically mounted into every pod running in the cluster, by k8s
// we convert this kubeconfig into a secret, and use that secret to connect

// for external clusters -> user needs to upload the kubeconfig for that cluster, we will alos convert that to a secret and use that to connect
// for managed clusters created by crossplane, we will use secret creatd by crossplane to connect to the cluster

type OnboardingService interface {
	Onboard(ctx context.Context, sub string, email string) (uuid.UUID, error)
}

type onboardingService struct {
	userRepo repository.UserRepository
	db       *pgxpool.Pool
	k8s      client.Client
	vCfg     *config.VulkanConfig
}

func NewOnboardingService(userRepo repository.UserRepository, db *pgxpool.Pool) OnboardingService {
	return &onboardingService{
		userRepo: userRepo,
		db:       db,
	}
}

func (s *onboardingService) Onboard(ctx context.Context, sub string, email string) (uuid.UUID, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx) // ensure rollback on error

	// create user
	userId, err := s.userRepo.InsertNewUserWithTx(ctx, tx, sub, email)
	if err != nil {
		tx.Rollback(ctx) // rollback if insert fails
		return uuid.Nil, err
	}
	// create default org and add user as admin
	oidOrg := uuid.New()
	defaultOrgDomain := "default-org.vulkan.io"
	orgName := "default-org"
	_, _ = tx.Exec(ctx, `INSERT INTO orgs(id, name, owner_id, domain) VALUES($1,$2,$3,$4)`, oidOrg, orgName, userId, defaultOrgDomain)
	_, _ = tx.Exec(ctx, `INSERT INTO org_members(user_id,org_id,role) VALUES($1,$2,'admin')`, userId, oidOrg, "admin")

	// create default project and add user as admin
	oidProj := uuid.New()
	_, _ = tx.Exec(ctx, `INSERT INTO projects(id,org_id,name) VALUES($1,$2,'default-proj')`, oidProj, oidOrg)
	_, _ = tx.Exec(ctx, `INSERT INTO project_members(user_id,project_id,role) VALUES($1,$2,'admin')`, userId, oidProj, "admin")

	// create default cluster record
	oidCluster := uuid.New()
	_, _ = tx.Exec(ctx, `INSERT INTO clusters(id,org_id,name,type,status) VALUES ($1,$2,'current-cluster','attached','ready')`, oidCluster, oidOrg)

	// many-to-many link  (project ⇄ cluster)
	_, _ = tx.Exec(ctx, `INSERT INTO project_clusters(project_id,cluster_id) VALUES ($1,$2)`, oidProj, oidCluster)

	tx.Commit(ctx) // commit if everything is fine

	// k8s crds

	// note -> kube config is a file that stores config info for accessing k8s clusters
	// we load a kube config into our cluster CR via a secret

	// ensureInClusterSecret -> ensures that the secret we use to load the kube config is present in the cluster
	err = ensureInClusterSecret(ctx, s.vCfg.DefaultAttachedClusterConnectionSecret, s.k8s)
	if err != nil {
		s.revertOnboarding(ctx, tx, userId, oidOrg, oidProj, oidCluster)
		return uuid.Nil, fmt.Errorf("failed to ensure in-cluster kubeconfig secret: %w", err)
	}

	// org
	if err := s.k8s.Create(ctx, &platformv1.Org{
		ObjectMeta: metav1.ObjectMeta{Name: oidOrg.String()},
		Spec:       platformv1.OrgSpec{DisplayName: "default-org"},
	}); err != nil {
		s.revertOnboarding(ctx, tx, userId, oidOrg, oidProj, oidCluster)
		return uuid.Nil, fmt.Errorf("failed to create org CRD: %w", err)
	}

	// cluster
	if err := s.k8s.Create(ctx, &platformv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   oidCluster.String(),
			Labels: map[string]string{"org": oidOrg.String()},
		},
		Spec: platformv1.ClusterSpec{
			Type:                 "attached",
			Region:               "",
			KubeconfigSecretName: s.vCfg.DefaultAttachedClusterConnectionSecret,
			OrgRef:               oidOrg.String(),
		},
	}); err != nil {
		s.revertOnboarding(ctx, tx, userId, oidOrg, oidProj, oidCluster)
		return uuid.Nil, fmt.Errorf("failed to create cluster CRD: %w", err)
	}

	// project
	if err := s.k8s.Create(ctx, &platformv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   oidProj.String(),
			Labels: map[string]string{"org": oidOrg.String()},
		},
		Spec: platformv1.ProjectSpec{DisplayName: "default-proj"},
	}); err != nil {
		s.revertOnboarding(ctx, tx, userId, oidOrg, oidProj, oidCluster)
		return uuid.Nil, fmt.Errorf("failed to create project CRD: %w", err)
	}

	// projectClusterBinding CRD (declares the many-to-many link)
	if err := s.k8s.Create(ctx, &platformv1.ProjectClusterBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", oidProj.String(), oidCluster.String()),
		},
		Spec: platformv1.ProjectClusterBindingSpec{
			ProjectRef: oidProj.String(),
			ClusterRef: oidCluster.String(),
		},
	}); err != nil {
		s.revertOnboarding(ctx, tx, userId, oidOrg, oidProj, oidCluster)
		return uuid.Nil, fmt.Errorf("failed to create cluster project binding CRD: %w", err)
	}

	return userId, nil
}

func (s *onboardingService) revertOnboarding(ctx context.Context, tx pgx.Tx, userId, orgID, projID, clusterID uuid.UUID) error {
	// simply delete user, cascade will take care of related records
	_, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userId)
	if err != nil {
		return err
	}

	// delete crds
	toDelete := []client.Object{
		&platformv1.Org{ObjectMeta: metav1.ObjectMeta{Name: orgID.String()}},
		&platformv1.Project{ObjectMeta: metav1.ObjectMeta{Name: projID.String()}},
		&platformv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: clusterID.String()}},
		&platformv1.ProjectClusterBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s", projID.String(), clusterID.String()),
			},
		},
	}

	for _, obj := range toDelete {
		err := s.k8s.Delete(ctx, obj)
		if processedErr := client.IgnoreNotFound(err); processedErr != nil {
			return fmt.Errorf("failed to delete CRD object %s %s: %w",
				obj.GetObjectKind().GroupVersionKind().Kind,
				obj.GetName(),
				processedErr)
		}
	}
	return nil
}

// ask Kubernetes for the “in-cluster” config using -> rest.InClusterConfig() returns the API-server URL, a bearer-token, and the CA bundle that every pod already mounts.
// wrap returned data in a kube-config YAML
// build a tiny kube-config that has one user/cluster/context block using the same token.
// create a Secret
func ensureInClusterSecret(ctx context.Context, secretName string, c client.Client) error {

	// Quick check – if the Secret already exists we’re done.
	var existing corev1.Secret
	err := c.Get(ctx, client.ObjectKey{Namespace: "default", Name: secretName}, &existing)
	if err == nil {
		return nil // secret already there
	}
	if client.IgnoreNotFound(err) != nil {
		return err // real error
	}

	// in-cluster rest.Config
	// InClusterConfig returns a config object which uses the service account kubernetes gives to pods.
	// it's intended for clients running inside a k8s pod.
	// returns ErrNotInCluster if called from a process not running in a kubernetes environment.
	ic, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// we use the service account token to create a kubeconfig file

	// convert rest.Config ➜ minimal kube-config YAML
	kc := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"in-cluster": {
				Server:                   ic.Host,
				CertificateAuthorityData: ic.CAData,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"sa": {Token: string(ic.BearerToken)},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"ctx": {Cluster: "in-cluster", AuthInfo: "sa"},
		},
		CurrentContext: "ctx",
	}
	kubeconfigYAML, err := clientcmd.Write(kc)
	if err != nil {
		return err
	}

	// create the in cluster Secret from the generated kube config file
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      secretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": kubeconfigYAML,
		},
	}
	return c.Create(ctx, &secret) // idempotent because we checked first
}
