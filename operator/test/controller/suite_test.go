package controller

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	platformv1alpha1 "github.com/mofe64/vulkan/operator/api/v1alpha1"
	"github.com/mofe64/vulkan/operator/internal/metrics"
	utils "github.com/mofe64/vulkan/operator/internal/utils"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx          context.Context
	cancel       context.CancelFunc
	testEnv      *envtest.Environment
	cfg          *rest.Config
	k8sClient    client.Client
	testRegistry *prometheus.Registry
	testDB       *sql.DB
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	err = platformv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// create a test registry for prom metrics
	testRegistry = prometheus.NewRegistry()
	testRegistry.MustRegister(
		metrics.ClustersPerOrg,
		metrics.ProjectsPerOrg,
		metrics.ApplicationsPerOrg,
		metrics.OrgQuotaUsage,
	)

	// create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "vulkan-test-db")
	Expect(err).NotTo(HaveOccurred())

	// connect to test database using sqlite
	testDB, err = utils.ConnectDB(filepath.Join(tempDir, "test.db"))
	Expect(err).NotTo(HaveOccurred())
	Expect(testDB).NotTo(BeNil())

	// run migrations
	_, err = testDB.ExecContext(ctx, `
		-- Create users table
		CREATE TABLE IF NOT EXISTS users (
			id         TEXT PRIMARY KEY,
			oidc_sub   TEXT UNIQUE NOT NULL,
			email      TEXT UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		-- Create projects table
		CREATE TABLE IF NOT EXISTS projects (
			id         TEXT PRIMARY KEY,
			org_id     TEXT NOT NULL,
			name       TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		-- Create project_members table
		CREATE TABLE IF NOT EXISTS project_members (
			user_id    TEXT NOT NULL,
			project_id TEXT NOT NULL,
			role       TEXT NOT NULL CHECK (role IN ('admin', 'maintainer', 'viewer')),
			PRIMARY KEY (user_id, project_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
		);
		
		-- Create indexes
		CREATE INDEX IF NOT EXISTS idx_project_members_user ON project_members(user_id);
		CREATE INDEX IF NOT EXISTS idx_project_members_project ON project_members(project_id);
	`)
	Expect(err).NotTo(HaveOccurred())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())

	// close the test database
	if testDB != nil {
		testDB.Close()
	}

})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
