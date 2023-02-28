package unleash

import (
	"context"
	"errors"

	admin "google.golang.org/api/sqladmin/v1beta4"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type Unleash struct {
	TeamName            string
	KubernetesNamespace string
	DatabaseInstance    *admin.DatabaseInstance
	Database            *admin.Database
	DatabaseUser        *admin.User
	Secret              *v1.Secret
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
