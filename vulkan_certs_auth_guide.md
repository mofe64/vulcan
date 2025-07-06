# Vulkan PaaS Control Plane

This README provides a high-level overview of the core infrastructure components responsible for secure traffic routing and authentication within the Vulkan PaaS.

## 1. TLS and Traffic Management (Cert-Manager, Ingress, NGINX Ingress Controller)

- **NGINX Ingress Controller:** Acts as the entry point for all external HTTP/HTTPS traffic into the Vulkan K8s cluster. It routes requests to the correct internal services (UI, API, Dex) based on the hostname and path. It also handles **TLS termination**, decrypting incoming HTTPS traffic.
- **Cert-Manager:** Automates the lifecycle of TLS certificates. It integrates with Certificate Authorities (like Let's Encrypt) to **automatically provision, renew, and manage** your SSL/TLS certificates.
- **Ingress Resource:** A Kubernetes object that defines how external traffic should be routed to internal services. It acts as a configuration interface for the NGINX Ingress Controller.

**How they relate:** Our Ingress resources for the UI, API, and Dex are configured with specific hostnames and tell the NGINX Ingress Controller where to send traffic. They also specify a `secretName` for TLS certificates and an annotation (`cert-manager.io/cluster-issuer: "letsencrypt-prod"`) that signals **Cert-Manager** to automatically obtain a certificate from Let's Encrypt and store it in that `secretName`. NGINX then uses this secret to serve traffic over HTTPS.

## 2. Application-Specific Usage

For our application components (UI, API, DEX):

- **DNS:** We use specific subdomains for each component:
  - `vulkan.strawhatengineer.com` (for the UI)
  - `api.vulkan.strawhatengineer.com` (for the API)
  - `dex.strawhatengineer.com` (for the Dex authentication service)
- **Ingress & TLS:** Each of these domains is exposed via its own Kubernetes Ingress resource. Cert-Manager, configured with a `ClusterIssuer` named `letsencrypt-prod`, automatically provisions and renews TLS certificates for these hostnames, ensuring all external communication is encrypted with HTTPS.
- **NGINX Ingress Controller** is the backbone handling the routing for all these domains.

For our k8s operator:

- We have a certificate names `metrics-certs` this is the TLS certificate for the metric server attached to the operator

- We also have a self-signed `Issuer` named `selfsigned-issuer` that is namespaced scoped to the installation namespace for vulkan, this self signed issuer is responsible for issuing the `metrics-certs` certificate for the operator metrics-server.

## 3. Dex Authentication Explained

**Dex** is an OpenID Connect (OIDC) identity provider. Its primary role is to act as a **federated identity broker**, allowing our applications (Vulkan UI and API) to authenticate users against external identity sources (like GitHub) without needing to implement direct integrations themselves.

**Components and Workflow:**

- **Issuer URL:** Dex serves its OIDC endpoints from `https://dex.vulkan.strawhatengineer.com/dex`.
- **Storage:** Dex uses Kubernetes as its storage backend (`inCluster: true`), managing its internal state within the cluster.
- **Static Clients:** These define the applications that will integrate with Dex:
  - `vulkan-ui`: A public OIDC client used by the frontend. It has a `redirectURI` configured (e.g., `https://vulkan.strawhatengineer.com/callback`) where Dex redirects the user after successful authentication.
  - `vulkan-api`: A confidential OIDC client used by the backend API. It has a `secret` (a shared key with Dex) for secure token exchange and its own `redirectURI`.
- **Connectors:** These define the external identity providers Dex can use. In our setup, we use a `github` connector, allowing users to log in with their GitHub accounts.
- **Authentication Flow:**
  1.  The **Vulkan UI** initiates an OIDC authentication request to Dex.
  2.  Dex redirects the user's browser to **GitHub** for login/consent.
  3.  GitHub authenticates the user and redirects them back to Dex.
  4.  Dex issues an OIDC token (a JWT) and redirects the user back to the **Vulkan UI**'s configured `redirectURI`.
  5.  The **Vulkan API** validates incoming JWTs by fetching Dex's public keys (JWKS) from `https://dex.vulkan.strawhatengineer.com/dex/keys` and verifying the token's signature and claims.

## 4. Other Important Facts

- **Helm Umbrella Chart:** The entire infrastructure (including Cert-Manager, NGINX Ingress Controller, Dex, NATS, etc., alongside the Vulkan UI and API) is deployed and managed as a single Helm umbrella chart. This simplifies deployment, configuration, and upgrades.
- **Secrets Management:** All sensitive credentials (e.g., Dex API client secrets, GitHub OAuth client ID/secret) are securely managed as Kubernetes Secrets and are never hardcoded directly in `values.yaml`. They are passed to the Helm chart via a separate `values.secrets.yaml` file.

---
