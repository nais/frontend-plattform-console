package unleash

import (
	"fmt"

	fqdnV1alpha3 "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	"github.com/nais/bifrost/pkg/config"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func boolRef(b bool) *bool {
	boolVar := b
	return &boolVar
}

func int64Ref(i int64) *int64 {
	intvar := i
	return &intvar
}

func newFQDNNetworkPolicySpec(teamName string, kubeNamespace string) fqdnV1alpha3.FQDNNetworkPolicy {
	protocolTCP := corev1.ProtocolTCP

	return fqdnV1alpha3.FQDNNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-fqdn", teamName),
			Namespace: kubeNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "FQDNNetworkPolicy",
			APIVersion: "networking.gke.io/v1alpha3",
		},
		Spec: fqdnV1alpha3.FQDNNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":   teamName,
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
							FQDNs: []string{"sqladmin.googleapis.com", "www.gstatic.com"},
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

func newUnleashSpec(
	c *config.Config,
	teamName string,
) unleashv1.Unleash {
	cloudSqlProto := corev1.ProtocolTCP
	cloudSqlPort := intstr.FromInt(3307)

	googleIapAudience := c.GoogleIAPAudience()

	return unleashv1.Unleash{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Unleash",
			APIVersion: "unleash.nais.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      teamName,
			Namespace: c.Unleash.InstanceNamespace,
		},
		Spec: unleashv1.UnleashSpec{
			Size: 1,
			Database: unleashv1.DatabaseConfig{
				Host:                  "localhost",
				Port:                  "5432",
				SSL:                   "false",
				SecretName:            teamName,
				SecretUserKey:         "POSTGRES_USER",
				SecretPassKey:         "POSTGRES_PASSWORD",
				SecretHostKey:         "POSTGRES_HOST",
				SecretDatabaseNameKey: "POSTGRES_DB",
			},
			WebIngress: unleashv1.IngressConfig{
				Enabled: true,
				Host:    fmt.Sprintf("%s-%s", teamName, c.Unleash.InstanceWebIngressHost),
				Path:    "/",
				Class:   c.Unleash.InstanceWebIngressClass,
			},
			ApiIngress: unleashv1.IngressConfig{
				Enabled: true,
				Host:    fmt.Sprintf("%s-%s", teamName, c.Unleash.InstanceAPIIngressHost),
				Path:    "/api",
				Class:   c.Unleash.InstanceAPIIngressClass,
			},
			NetworkPolicy: unleashv1.NetworkPolicyConfig{
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
				},
			},
			ExtraEnvVars: []corev1.EnvVar{{
				Name:  "GOOGLE_IAP_AUDIENCE",
				Value: googleIapAudience,
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
			}},
			ExistingServiceAccountName: c.Unleash.InstanceServiceaccount,
			Resources:                  corev1.ResourceRequirements{},
		},
	}
}
