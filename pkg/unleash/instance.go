package unleash

import (
	"context"
	"fmt"
	"time"

	fqdnV1alpha3 "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	"github.com/nais/bifrost/pkg/config"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	admin "google.golang.org/api/sqladmin/v1beta4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type UnleashInstance struct {
	Name                 string
	KubernetesNamespace  string
	CreatedAt            metav1.Time
	ServerInstance       *unleashv1.Unleash
	DatabaseInstanceName string
	DatabaseProjectName  string
	Database             *admin.Database
	DatabaseUser         *admin.User
	DatabaseSecret       *corev1.Secret
}

func NewUnleashInstance(serverInstance *unleashv1.Unleash) *UnleashInstance {
	return &UnleashInstance{
		Name:                serverInstance.ObjectMeta.Name,
		KubernetesNamespace: serverInstance.ObjectMeta.Namespace,
		CreatedAt:           serverInstance.ObjectMeta.CreationTimestamp,
		ServerInstance:      serverInstance,
	}
}

func humanReadableAge(age metav1.Time) string {
	now := time.Now()
	diff := now.Sub(age.Time)

	if diff.Hours() < 24 {
		return "less than a day"
	} else if diff.Hours() < 24*7 {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	} else if diff.Hours() < 24*30 {
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	} else if diff.Hours() < 24*365 {
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", months)
	} else {
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year"
		}
		return fmt.Sprintf("%d years", years)
	}
}

func (u UnleashInstance) Age() string {
	return humanReadableAge(u.CreatedAt)
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
	return u.ServerInstance.IsReady()
}

func (u *UnleashInstance) Status() string {
	if u.ServerInstance != nil {
		if u.ServerInstance.IsReady() {
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
		if u.ServerInstance.IsReady() {
			return "green"
		} else {
			return "red"
		}
	} else {
		return "orange"
	}
}

func (u *UnleashInstance) GetDatabase(ctx context.Context, client *admin.DatabasesService) error {
	database, err := getDatabase(ctx, client, u.DatabaseInstanceName, u.DatabaseProjectName, u.Name)
	if err != nil {
		return err
	}

	u.Database = database

	return nil
}

func (u *UnleashInstance) GetDatabaseUser(ctx context.Context, client *admin.UsersService) error {
	user, err := getDatabaseUser(ctx, client, u.DatabaseInstanceName, u.DatabaseProjectName, u.Name)
	if err != nil {
		return err
	}

	u.DatabaseUser = user

	return nil
}

func getServer(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, name string) (*unleashv1.Unleash, error) {
	unleashDefinition := unleashv1.Unleash{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: kubeNamespace}}
	err := kubeClient.Get(ctx, ctrl.ObjectKeyFromObject(&unleashDefinition), &unleashDefinition)
	if err != nil {
		return nil, err
	}
	return &unleashDefinition, nil
}

func deleteServer(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, name string) error {
	unleashDefinition := unleashv1.Unleash{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: kubeNamespace}}
	return kubeClient.Delete(ctx, &unleashDefinition)
}

func createServer(ctx context.Context, kubeClient ctrl.Client, config *config.Config, uc *UnleashConfig) error {
	unleashDefinition := UnleashDefinition(config, uc)
	return kubeClient.Create(ctx, &unleashDefinition)
}

func updateServer(ctx context.Context, kubeClient ctrl.Client, config *config.Config, uc *UnleashConfig) error {
	unleashDefinitionOld, err := getServer(ctx, kubeClient, config.Unleash.InstanceNamespace, uc.Name)
	if err != nil {
		return err
	}

	unleashDefinitionNew := UnleashDefinition(config, uc)
	unleashDefinitionNew.ObjectMeta.ResourceVersion = unleashDefinitionOld.ObjectMeta.ResourceVersion

	return kubeClient.Update(ctx, &unleashDefinitionNew)
}

func getFQDNNetworkPolicy(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, name string) (*fqdnV1alpha3.FQDNNetworkPolicy, error) {
	fqdn := fqdnV1alpha3.FQDNNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-fqdn", name), Namespace: kubeNamespace}}
	err := kubeClient.Get(ctx, ctrl.ObjectKeyFromObject(&fqdn), &fqdn)
	if err != nil {
		return nil, err
	}
	return &fqdn, nil
}

func deleteFQDNNetworkPolicy(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, name string) error {
	fqdn := fqdnV1alpha3.FQDNNetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-fqdn", name), Namespace: kubeNamespace}}
	return kubeClient.Delete(ctx, &fqdn)
}

func createFQDNNetworkPolicy(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, name string) error {
	fqdn := FQDNNetworkPolicyDefinition(name, kubeNamespace)
	return kubeClient.Create(ctx, &fqdn)
}

func updateFQDNNetworkPolicy(ctx context.Context, kubeClient ctrl.Client, kubeNamespace string, name string) error {
	fqdnOld, err := getFQDNNetworkPolicy(ctx, kubeClient, kubeNamespace, name)
	if err != nil {
		return err
	}

	fqdnNew := FQDNNetworkPolicyDefinition(name, kubeNamespace)
	fqdnNew.ObjectMeta.ResourceVersion = fqdnOld.ObjectMeta.ResourceVersion

	return kubeClient.Update(ctx, &fqdnNew)
}
