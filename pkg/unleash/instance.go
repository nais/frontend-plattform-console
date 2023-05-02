package unleash

import (
	"context"

	fqdnV1alpha3 "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	"github.com/nais/bifrost/pkg/config"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	admin "google.golang.org/api/sqladmin/v1beta4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type UnleashInstance struct {
	TeamName            string
	KubernetesNamespace string
	CreatedAt           metav1.Time
	serverInstance      *unleashv1.Unleash
	databaseInstance    *admin.DatabaseInstance
	database            *admin.Database
	databaseUser        *admin.User
	secret              *corev1.Secret
}

func NewUnleashInstance(serverInstance *unleashv1.Unleash) *UnleashInstance {
	return &UnleashInstance{
		TeamName:            serverInstance.ObjectMeta.Name,
		KubernetesNamespace: serverInstance.ObjectMeta.Namespace,
		CreatedAt:           serverInstance.ObjectMeta.CreationTimestamp,
		serverInstance:      serverInstance,
	}
}

func (u *UnleashInstance) GetDatabaseUser(ctx context.Context, client *admin.Service) error {
	user, err := getDatabaseUser(ctx, client, u.databaseInstance, u.database.Name)
	if err != nil {
		return err
	}

	u.databaseUser = user

	return nil
}

func deleteServer(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, teamName string) error {
	unleashDefinition := unleashv1.Unleash{ObjectMeta: metav1.ObjectMeta{Name: teamName, Namespace: kubeNamespace}}
	return kubeClient.Delete(ctx, &unleashDefinition)
}

func createServer(ctx context.Context, kubeClient ctrl.Client, config *config.Config, teamName string) error {
	unleashDefinition := newUnleashSpec(config, teamName)
	return kubeClient.Create(ctx, &unleashDefinition)
}

func deleteFQDNNetworkPolicy(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, teamName string) error {
	fqdn := fqdnV1alpha3.FQDNNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: teamName, Namespace: kubeNamespace}}
	return kubeClient.Delete(ctx, &fqdn)
}

func createFQDNNetworkPolicy(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, teamName string) error {
	fqdn := newFQDNNetworkPolicySpec(teamName, kubeNamespace)
	return kubeClient.Create(ctx, &fqdn)
}
