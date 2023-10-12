package unleash

import (
	"fmt"
	"strings"

	fqdnV1alpha3 "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/utils"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	UnleashCustomImageRepo = "europe-north1-docker.pkg.dev/nais-io/nais/images/"
	UnleashCustomImageName = "unleash-v4"
	UnleashRequestCPU      = "100m"
	UnleashRequestMemory   = "128Mi"
	UnleashLimitMemory     = "256Mi"
	SqlProxyRequestCPU     = "10m"
	SqlProxyRequestMemory  = "100Mi"
	SqlProxyLimitMemory    = "100Mi"
)

func boolRef(b bool) *bool {
	boolVar := b
	return &boolVar
}

func int64Ref(i int64) *int64 {
	intvar := i
	return &intvar
}

func FQDNNetworkPolicyDefinition(name string, kubeNamespace string) fqdnV1alpha3.FQDNNetworkPolicy {
	protocolTCP := corev1.ProtocolTCP

	return fqdnV1alpha3.FQDNNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-fqdn", name),
			Namespace: kubeNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "FQDNNetworkPolicy",
			APIVersion: "networking.gke.io/v1alpha3",
		},
		Spec: fqdnV1alpha3.FQDNNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":   name,
					"app.kubernetes.io/part-of":    "unleasherator",
					"app.kubernetes.io/name":       "Unleash",
					"app.kubernetes.io/created-by": "controller-manager",
				},
			},
			Egress: []fqdnV1alpha3.FQDNNetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 443},
							Protocol: &protocolTCP,
						},
					},
					To: []fqdnV1alpha3.FQDNNetworkPolicyPeer{
						{
							FQDNs: []string{"sqladmin.googleapis.com", "www.gstatic.com", "hooks.slack.com"},
						},
					},
				},
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 80},
							Protocol: &protocolTCP,
						},
						{
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 988},
							Protocol: &protocolTCP,
						},
					},
					To: []fqdnV1alpha3.FQDNNetworkPolicyPeer{
						{
							FQDNs: []string{"metadata.google.internal"},
						},
					},
				},
			},
		},
	}
}

func customImageForVersion(customVersion string) string {
	return fmt.Sprintf("%s%s:%s", UnleashCustomImageRepo, UnleashCustomImageName, customVersion)
}

func versionFromImage(image string) string {
	return strings.Split(image, ":")[1]
}

func getServerEnvVar(server *unleashv1.Unleash, name, defaultValue string, returnDefault bool) string {
	for _, envVar := range server.Spec.ExtraEnvVars {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	if returnDefault {
		return defaultValue
	} else {
		return ""
	}
}

type UnleashConfig struct {
	Name              string
	CustomVersion     string
	EnableFederation  bool
	FederationNonce   string
	AllowedTeams      string
	AllowedNamespaces string
	AllowedClusters   string
	LogLevel          string
}

func UnleashVariables(server *unleashv1.Unleash, returnDefaults bool) *UnleashConfig {
	uc := &UnleashConfig{}

	uc.Name = server.GetName()

	if server.Spec.CustomImage != "" {
		uc.CustomVersion = versionFromImage(server.Spec.CustomImage)
	}

	uc.AllowedTeams = getServerEnvVar(server, "TEAMS_ALLOWED_TEAMS", uc.Name, returnDefaults)
	uc.LogLevel = getServerEnvVar(server, "LOG_LEVEL", "warn", returnDefaults)
	uc.EnableFederation = server.Spec.Federation.Enabled

	uc.AllowedNamespaces = utils.JoinNoEmpty(server.Spec.Federation.Namespaces, ",")
	if uc.AllowedNamespaces == "" {
		uc.AllowedNamespaces = getServerEnvVar(server, "TEAMS_ALLOWED_NAMESPACES", "", returnDefaults)
	}

	uc.AllowedClusters = utils.JoinNoEmpty(server.Spec.Federation.Clusters, ",")
	if uc.AllowedClusters == "" {
		uc.AllowedClusters = getServerEnvVar(server, "TEAMS_ALLOWED_CLUSTERS", "", returnDefaults)
	}

	return uc
}

func UnleashDefinition(
	c *config.Config,
	uc *UnleashConfig,
) unleashv1.Unleash {
	cloudSqlProto := corev1.ProtocolTCP
	cloudSqlPort := intstr.FromInt(3307)

	teamsApiProto := corev1.ProtocolTCP
	teamsApiPort := intstr.FromInt(3000)
	teamsApiNamespace := "nais-system"
	teamsApiName := "teams-backend"

	googleIapAudience := c.GoogleIAPAudience()

	federationNonce := uc.FederationNonce
	if federationNonce == "" {
		federationNonce = utils.RandomString(8)
	}

	server := unleashv1.Unleash{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Unleash",
			APIVersion: "unleash.nais.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      uc.Name,
			Namespace: c.Unleash.InstanceNamespace,
		},
		Spec: unleashv1.UnleashSpec{
			Size: 1,
			Database: unleashv1.UnleashDatabaseConfig{
				Host:                  "localhost",
				Port:                  "5432",
				SSL:                   "false",
				SecretName:            uc.Name,
				SecretUserKey:         "POSTGRES_USER",
				SecretPassKey:         "POSTGRES_PASSWORD",
				SecretDatabaseNameKey: "POSTGRES_DB",
			},
			WebIngress: unleashv1.UnleashIngressConfig{
				Enabled: true,
				Host:    fmt.Sprintf("%s-%s", uc.Name, c.Unleash.InstanceWebIngressHost),
				Path:    "/",
				Class:   c.Unleash.InstanceWebIngressClass,
			},
			ApiIngress: unleashv1.UnleashIngressConfig{
				Enabled: true,
				Host:    fmt.Sprintf("%s-%s", uc.Name, c.Unleash.InstanceAPIIngressHost),
				// Allow access to /health endpoint, change to /api when https://github.com/nais/unleasherator/issues/100 is resolved
				Path:  "/",
				Class: c.Unleash.InstanceAPIIngressClass,
			},
			NetworkPolicy: unleashv1.UnleashNetworkPolicyConfig{
				Enabled:  true,
				AllowDNS: true,
				ExtraEgressRules: []networkingv1.NetworkPolicyEgressRule{
					{
						Ports: []networkingv1.NetworkPolicyPort{{
							Protocol: &cloudSqlProto,
							Port:     &cloudSqlPort,
						}},
						To: []networkingv1.NetworkPolicyPeer{{
							IPBlock: &networkingv1.IPBlock{
								CIDR: fmt.Sprintf("%s/32", c.Unleash.SQLInstanceAddress),
							},
						}},
					},
					{
						Ports: []networkingv1.NetworkPolicyPort{{
							Protocol: &teamsApiProto,
							Port:     &teamsApiPort,
						}},
						To: []networkingv1.NetworkPolicyPeer{{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": teamsApiNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": teamsApiName,
								},
							},
						}},
					},
				},
			},
			Federation: unleashv1.UnleashFederationConfig{
				Enabled:     uc.EnableFederation,
				Namespaces:  utils.SplitNoEmpty(uc.AllowedNamespaces, ","),
				Clusters:    utils.SplitNoEmpty(uc.AllowedClusters, ","),
				SecretNonce: federationNonce,
			},
			ExtraEnvVars: []corev1.EnvVar{{
				Name:  "GOOGLE_IAP_AUDIENCE",
				Value: googleIapAudience,
			}, {
				Name:  "TEAMS_API_URL",
				Value: c.Unleash.TeamsApiURL,
			}, {
				Name: "TEAMS_API_TOKEN",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: c.Unleash.TeamsApiSecretName,
						},
						Key: c.Unleash.TeamsApiSecretTokenKey,
					},
				},
			}, {
				Name:  "TEAMS_ALLOWED_TEAMS",
				Value: uc.AllowedTeams,
			}, {
				Name:  "TEAMS_ALLOWED_NAMESPACES",
				Value: uc.AllowedNamespaces,
			}, {
				Name:  "TEAMS_ALLOWED_CLUSTERS",
				Value: uc.AllowedClusters,
			}, {
				Name:  "LOG_LEVEL",
				Value: uc.LogLevel,
			}},
			ExtraContainers: []corev1.Container{{
				Name:  "sql-proxy",
				Image: c.CloudConnectorProxy,
				Args: []string{
					"--structured-logs",
					"--port=5432",
					fmt.Sprintf("%s:%s:%s", c.Google.ProjectID,
						c.Unleash.SQLInstanceRegion,
						c.Unleash.SQLInstanceID),
				},
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
					Privileged:               boolRef(false),
					RunAsUser:                int64Ref(65532),
					RunAsNonRoot:             boolRef(true),
					AllowPrivilegeEscalation: boolRef(false),
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(SqlProxyRequestCPU),
						corev1.ResourceMemory: resource.MustParse(SqlProxyRequestMemory),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse(SqlProxyLimitMemory),
					},
				},
			}},
			ExistingServiceAccountName: c.Unleash.InstanceServiceaccount,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(UnleashRequestCPU),
					corev1.ResourceMemory: resource.MustParse(UnleashRequestMemory),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse(UnleashLimitMemory),
				},
			},
		},
	}

	if uc.CustomVersion != "" {
		server.Spec.CustomImage = customImageForVersion(uc.CustomVersion)
	}

	return server
}
