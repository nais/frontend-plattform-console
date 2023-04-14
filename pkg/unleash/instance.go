package unleash

import (
	"context"
	"errors"
	"fmt"

	fqdnV1alpha3 "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	"github.com/nais/bifrost/pkg/config"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	admin "google.golang.org/api/sqladmin/v1beta4"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_config "sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Unleash struct {
	TeamName            string
	KubernetesNamespace string
	DatabaseInstance    *admin.DatabaseInstance
	Database            *admin.Database
	DatabaseUser        *admin.User
	Secret              *corev1.Secret
}

func (u *Unleash) GetDatabaseUser(ctx context.Context, client *admin.Service) error {
	user, err := getDatabaseUser(ctx, client, u.DatabaseInstance, u.Database.Name)
	if err != nil {
		return err
	}

	u.DatabaseUser = user

	return nil
}

func (u *Unleash) Delete(ctx context.Context, googleClient *admin.Service, kubeClient ctrl.Client) error {
	dbUserErr := deleteDatabaseUser(ctx, googleClient, u.DatabaseInstance, u.Database.Name)
	dbErr := deleteDatabase(ctx, googleClient, u.DatabaseInstance, u.Database.Name)
	dbUserSecretErr := deleteDatabaseUserSecret(ctx, kubeClient, u.KubernetesNamespace, u.Database.Name)

	return errors.Join(dbUserErr, dbErr, dbUserSecretErr)
}

func GetInstances(ctx context.Context, googleClient *admin.Service, databaseInstance *admin.DatabaseInstance, kubeNamespace string) ([]Unleash, error) {
	databases, err := getDatabases(ctx, googleClient, databaseInstance)
	if err != nil {
		return nil, err
	}

	var instances []Unleash

	for _, database := range databases {
		if database.Name == "postgres" {
			continue
		}

		instances = append(instances, Unleash{
			TeamName:            database.Name,
			KubernetesNamespace: kubeNamespace,
			DatabaseInstance:    databaseInstance,
			Database:            database,
		})
	}

	return instances, nil
}

func GetInstance(ctx context.Context, googleClient *admin.Service, databaseInstance *admin.DatabaseInstance, databaseName string, kubeNamespace string) (Unleash, error) {
	database, err := getDatabase(ctx, googleClient, databaseInstance, databaseName)
	if err != nil {
		return Unleash{}, err
	}

	return Unleash{
		TeamName:            database.Name,
		KubernetesNamespace: kubeNamespace,
		DatabaseInstance:    databaseInstance,
		Database:            database,
	}, nil
}

func boolRef(b bool) *bool {
	boolVar := b
	return &boolVar
}

func int64Ref(i int64) *int64 {
	intvar := i
	return &intvar
}

func createUnleashSpec(
	bifrostConfig *config.Config,
	teamName string,
	googleIapAudience string,
) unleashv1.Unleash {
	tcpProtocol := "TCP"
	cloudSql := intstr.FromInt(3307)
	googleMetaDataPort := intstr.FromInt(988)
	port80 := intstr.FromInt(80)

	spec := unleashv1.UnleashSpec{
		Size: 0,
		Database: unleashv1.DatabaseConfig{
			SecretName:            teamName,
			SecretUserKey:         "POSTGRES_USER",
			SecretPassKey:         "POSTGRES_PASSWORD",
			SecretHostKey:         "POSTGRES_HOST",
			SecretDatabaseNameKey: "POSTGRES_DB",
		},
		WebIngress: unleashv1.IngressConfig{
			Enabled: true,
			Host:    fmt.Sprintf("%s-%s", teamName, bifrostConfig.Unleash.InstanceWebIngressHost),
			Path:    "/",
			Class:   bifrostConfig.Unleash.InstanceWebIngressClass,
		},
		ApiIngress: unleashv1.IngressConfig{
			Enabled: true,
			Host:    fmt.Sprintf("%s-%s", teamName, bifrostConfig.Unleash.InstanceAPIIngressHost),
			Path:    "/api",
			Class:   bifrostConfig.Unleash.InstanceAPIIngressClass,
		},
		NetworkPolicy: unleashv1.NetworkPolicyConfig{
			Enabled:  true,
			AllowDNS: true,
			ExtraEgressRules: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{
						Protocol: (*corev1.Protocol)(&tcpProtocol),
						Port:     &cloudSql,
					}},
					To: []networkingv1.NetworkPolicyPeer{{
						IPBlock: &networkingv1.IPBlock{
							CIDR: fmt.Sprintf("%s/32", bifrostConfig.Unleash.SQLInstanceAddress),
						},
					}},
				},
				{ // v these are google meta data servers
					Ports: []networkingv1.NetworkPolicyPort{{
						Protocol: (*corev1.Protocol)(&tcpProtocol),
						Port:     &googleMetaDataPort,
					}},
					To: []networkingv1.NetworkPolicyPeer{{
						IPBlock: &networkingv1.IPBlock{
							CIDR: "169.254.169.252/32",
						},
					}},
				},
				{
					Ports: []networkingv1.NetworkPolicyPort{{
						Protocol: (*corev1.Protocol)(&tcpProtocol),
						Port:     &googleMetaDataPort,
					}},
					To: []networkingv1.NetworkPolicyPeer{{
						IPBlock: &networkingv1.IPBlock{
							CIDR: "127.0.0.1/32",
						},
					}},
				},
				{
					Ports: []networkingv1.NetworkPolicyPort{{
						Protocol: (*corev1.Protocol)(&tcpProtocol),
						Port:     &port80,
					}},
					To: []networkingv1.NetworkPolicyPeer{{
						IPBlock: &networkingv1.IPBlock{
							CIDR: "169.254.169.254/32",
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
			Image: bifrostConfig.CloudConnectorProxy,
			Args: []string{
				"--structured-logs",
				"--port=5432",
				fmt.Sprintf("%s:%s:%s", bifrostConfig.Google.ProjectID,
					bifrostConfig.Unleash.SQLInstanceRegion,
					bifrostConfig.Unleash.SQLInstanceID),
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
		ExistingServiceAccountName: bifrostConfig.Unleash.InstanceServiceaccount,
		Resources:                  corev1.ResourceRequirements{},
	}

	return unleashv1.Unleash{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Unleash",
			APIVersion: "unleash.nais.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      teamName,
			Namespace: bifrostConfig.Unleash.InstanceNamespace,
		},
		Spec: spec,
	}
}

func createUnleashCrd(ctx context.Context, kubeClient ctrl.Client, config *config.Config, unleashDefinition unleashv1.Unleash, databaseName string, iapAudience string) error {
	schema := runtime.NewScheme()
	unleashv1.AddToScheme(schema)
	opts := ctrl.Options{
		Scheme: schema,
	}
	c, err := ctrl.New(ctrl_config.GetConfigOrDie(), opts)
	if err != nil {
		return err
	}

	err = c.Create(ctx, &unleashDefinition)
	if err != nil {
		return err
	}
	return nil
}

func CreateInstance(ctx context.Context,
	googleClient *admin.Service,
	databaseInstance *admin.DatabaseInstance,
	databaseName string,
	config *config.Config,
	kubeClient ctrl.Client,
) error {
	iapAudience := fmt.Sprintf("/projects/%s/global/backendServices/%s", config.Google.ProjectID, config.Google.IAPBackendServiceID)

	database, dbErr := createDatabase(ctx, googleClient, databaseInstance, databaseName)
	databaseUser, dbUserErr := createDatabaseUser(ctx, googleClient, databaseInstance, databaseName)
	secretErr := createDatabaseUserSecret(ctx, kubeClient, config.Unleash.InstanceNamespace, databaseInstance, database, databaseUser)
	fqdnCreationError := createFQDNNetworkPolicy(ctx, kubeClient, config.Unleash.InstanceNamespace, database.Name)
	unleashSpec := createUnleashSpec(config, databaseName, iapAudience)
	createCrdError := createUnleashCrd(ctx, kubeClient, config, unleashSpec, databaseName, iapAudience)
	if err := errors.Join(dbErr, dbUserErr, secretErr, fqdnCreationError, createCrdError); err != nil {
		return err
	}
	return nil
}

func createFQDNNetworkPolicy(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, teamName string) error {
	protocolTCP := corev1.ProtocolTCP

	typeMeta := metav1.TypeMeta{
		Kind:       "FQDNNetworkPolicy",
		APIVersion: "networking.gke.io/v1alpha3",
	}

	fqdn := fqdnV1alpha3.FQDNNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      teamName,
			Namespace: kubeNamespace,
		},
		TypeMeta: typeMeta,
		Spec: fqdnV1alpha3.FQDNNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/instance":   teamName,
					"app.kubernetes.io/part-of":    "unleasherator",
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
			},
		},
	}
	schema := runtime.NewScheme()
	fqdnV1alpha3.AddToScheme(schema)
	opts := ctrl.Options{
		Scheme: schema,
	}
	c, err := ctrl.New(ctrl_config.GetConfigOrDie(), opts)
	if err != nil {
		return err
	}

	err = c.Create(ctx, &fqdn)
	if err != nil {
		return err
	}
	return nil
}
