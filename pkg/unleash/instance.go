package unleash

import (
	"context"
	"fmt"

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
	ServerInstance      *unleashv1.Unleash
	DatabaseInstance    *admin.DatabaseInstance
	Database            *admin.Database
	DatabaseUser        *admin.User
	DatabaseSecret      *corev1.Secret
}

func NewUnleashInstance(serverInstance *unleashv1.Unleash) *UnleashInstance {
	return &UnleashInstance{
		TeamName:            serverInstance.ObjectMeta.Name,
		KubernetesNamespace: serverInstance.ObjectMeta.Namespace,
		CreatedAt:           serverInstance.ObjectMeta.CreationTimestamp,
		ServerInstance:      serverInstance,
	}
}

func (u *UnleashInstance) ApiUrl() string {
	if u.ServerInstance != nil {
		return fmt.Sprintf("https://%s/api/", u.ServerInstance.Spec.ApiIngress.Host)
	} else {
		return ""
	}
}

func (u *UnleashInstance) WebUrl() string {
	if u.ServerInstance != nil {
		return fmt.Sprintf("https://%s/", u.ServerInstance.Spec.WebIngress.Host)
	} else {
		return ""
	}
}

func (u *UnleashInstance) IsReady() bool {
	return u.ServerInstance.Status.IsReady()
}

func (u *UnleashInstance) Status() string {
	if u.ServerInstance != nil {
		if u.ServerInstance.Status.IsReady() {
			return "Ready"
		} else {
			return "Not ready"
		}
	} else {
		return "Status unknown"
	}
}

func (u *UnleashInstance) StatusLabel() string {
	if u.ServerInstance != nil {
		if u.ServerInstance.Status.IsReady() {
			return "green"
		} else {
			return "red"
		}
	} else {
		return "orange"
	}
}

func (u *UnleashInstance) GetDatabase(ctx context.Context, client *admin.Service) error {
	database, err := getDatabase(ctx, client, u.DatabaseInstance, u.TeamName)
	if err != nil {
		return err
	}

	u.Database = database

	return nil
}

func (u *UnleashInstance) GetDatabaseUser(ctx context.Context, client *admin.Service) error {
	user, err := getDatabaseUser(ctx, client, u.DatabaseInstance, u.TeamName)
	if err != nil {
		return err
	}

	u.DatabaseUser = user

	return nil
}

func deleteServer(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, teamName string) error {
	unleashDefinition := unleashv1.Unleash{ObjectMeta: metav1.ObjectMeta{Name: teamName, Namespace: kubeNamespace}}
	return kubeClient.Delete(ctx, &unleashDefinition)
}

func createServer(ctx context.Context, kubeClient ctrl.Client, config *config.Config, teamName string) error {
	unleashDefinition := NewUnleashSpec(config, teamName)
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
