package unleash

import (
	"context"
	"errors"

	"github.com/nais/bifrost/pkg/config"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	"github.com/sirupsen/logrus"
	admin "google.golang.org/api/sqladmin/v1beta4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type IUnleashService interface {
	List(ctx context.Context) ([]*UnleashInstance, error)
	Get(ctx context.Context, name string) (*UnleashInstance, error)
	Create(ctx context.Context, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters string) error
	Update(ctx context.Context, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters string) error
	Delete(ctx context.Context, name string) error
}

type UnleashService struct {
	googleClient *admin.Service
	sqlInstance  *admin.DatabaseInstance
	kubeClient   ctrl.Client
	config       *config.Config
	logger       *logrus.Logger
}

func NewUnleashService(googleClient *admin.Service, kubeClient ctrl.Client, sqlInstance *admin.DatabaseInstance, config *config.Config, logger *logrus.Logger) *UnleashService {
	return &UnleashService{
		googleClient: googleClient,
		sqlInstance:  sqlInstance,
		kubeClient:   kubeClient,
		config:       config,
		logger:       logger,
	}
}

func (s *UnleashService) List(ctx context.Context) ([]*UnleashInstance, error) {
	instanceList := []*UnleashInstance{}

	serverList := unleashv1.UnleashList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UnleashList",
			APIVersion: "unleasherator.nais.io/v1",
		},
	}

	opts := ctrl.ListOptions{
		Namespace: s.config.Unleash.InstanceNamespace,
	}

	if err := s.kubeClient.List(ctx, &serverList, &opts); err != nil {
		return nil, err
	}

	for _, instance := range serverList.Items {
		instanceList = append(instanceList, NewUnleashInstance(&instance))
	}

	return instanceList, nil
}

func (s *UnleashService) Get(ctx context.Context, name string) (*UnleashInstance, error) {
	serverInstance := &unleashv1.Unleash{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Unleash",
			APIVersion: "unleasherator.nais.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.config.Unleash.InstanceNamespace,
		},
	}

	if err := s.kubeClient.Get(ctx, ctrl.ObjectKeyFromObject(serverInstance), serverInstance); err != nil {
		return nil, err
	}

	return NewUnleashInstance(serverInstance), nil
}

func (s *UnleashService) Create(ctx context.Context, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters string) error {
	database, dbErr := createDatabase(ctx, s.googleClient, s.sqlInstance, name)
	databaseUser, dbUserErr := createDatabaseUser(ctx, s.googleClient, s.sqlInstance, name)
	secretErr := createDatabaseUserSecret(ctx, s.kubeClient, s.config.Unleash.InstanceNamespace, s.sqlInstance, database, databaseUser)
	fqdnCreationError := createFQDNNetworkPolicy(ctx, s.kubeClient, s.config.Unleash.InstanceNamespace, database.Name)
	createServerError := createServer(ctx, s.kubeClient, s.config, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters)

	if err := errors.Join(dbErr, dbUserErr, secretErr, fqdnCreationError, createServerError); err != nil {
		return err
	}
	return nil
}

func (s *UnleashService) Update(ctx context.Context, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters string) error {
	server := UnleashSpec(s.config, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters)
	return s.kubeClient.Update(ctx, &server)
}

func (s *UnleashService) Delete(ctx context.Context, name string) error {
	serverErr := deleteServer(ctx, s.kubeClient, s.config.Unleash.InstanceNamespace, name)
	netPolErr := deleteFQDNNetworkPolicy(ctx, s.kubeClient, s.config.Unleash.InstanceNamespace, name)
	dbUserSecretErr := deleteDatabaseUserSecret(ctx, s.kubeClient, s.config.Unleash.InstanceNamespace, name)
	dbErr := deleteDatabase(ctx, s.googleClient, s.sqlInstance, name)
	dbUserErr := deleteDatabaseUser(ctx, s.googleClient, s.sqlInstance, name)

	return errors.Join(serverErr, netPolErr, dbUserSecretErr, dbUserErr, dbErr)
}
