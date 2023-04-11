package unleash

import (
	"context"
	"errors"

	"github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	admin "google.golang.org/api/sqladmin/v1beta4"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
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

func (u *Unleash) Delete(ctx context.Context, googleClient *admin.Service, kubeClient *kubernetes.Clientset) error {
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

func CreateInstance(ctx context.Context, googleClient *admin.Service, databaseInstance *admin.DatabaseInstance, databaseName string, kubeClient *kubernetes.Clientset, kubeNamespace string) (Unleash, error) {
	database, dbErr := createDatabase(ctx, googleClient, databaseInstance, databaseName)
	databaseUser, dbUserErr := createDatabaseUser(ctx, googleClient, databaseInstance, databaseName)
	_, secretErr := createDatabaseUserSecret(ctx, kubeClient, kubeNamespace, databaseInstance, database, databaseUser)

	if err := errors.Join(dbErr, dbUserErr, secretErr); err != nil {
		return Unleash{}, err
	}

	// TODO: create kubernetes secret
	// TODO: create unleash instance

	return Unleash{
		TeamName:         databaseName,
		DatabaseInstance: databaseInstance,
		Database:         database,
		DatabaseUser:     databaseUser,
	}, nil
}

func createFQDNNetworkPolicy(ctx context.Context, kubeClient *kubernetes.Clientset, kubeNamespace string, databaseInstance *admin.DatabaseInstance) error {
	protocolTCP := corev1.ProtocolTCP

	fqdn := v1alpha3.FQDNNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unleash",
			Namespace: kubeNamespace,
		},
		Spec: v1alpha3.FQDNNetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "unleash",
				},
			},
			Egress: []v1alpha3.FQDNNetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 5432},
							Protocol: &protocolTCP,
						},
					},
					To: []v1alpha3.FQDNNetworkPolicyPeer{
						{
							FQDNs: []string{"sqladmin.googleapis.com"},
						},
					},
				},
			},
		},
	}
}
