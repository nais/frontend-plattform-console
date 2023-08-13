package unleash

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	admin "google.golang.org/api/sqladmin/v1beta4"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func createDatabase(ctx context.Context, client ISQLDatabasesService, projectName, instanceName, databaseName string) (*admin.Database, error) {
	database := &admin.Database{
		Name: databaseName,
	}

	_, err := client.Insert(projectName, instanceName, database).Context(ctx).Do()
	if err != nil {
		return database, &UnleashError{Err: err, Reason: "failed to create database"}
	}

	return database, nil
}

func getDatabaseUser(ctx context.Context, client ISQLUsersService, projectName, instanceName, databaseName string) (*admin.User, error) {
	user, err := client.Get(projectName, instanceName, databaseName).Context(ctx).Do()
	if err != nil {
		return user, &UnleashError{Err: err, Reason: "failed to get database user"}
	}

	return user, nil
}

func createDatabaseUser(ctx context.Context, client ISQLUsersService, projectName, instanceName, databaseName string) (*admin.User, error) {
	password, err := randomPassword(16)
	if err != nil {
		return nil, err
	}

	user := &admin.User{
		Name:     databaseName,
		Password: password,
	}

	_, err = client.Insert(projectName, instanceName, user).Context(ctx).Do()
	if err != nil {
		return user, &UnleashError{Err: err, Reason: "failed to create database user"}
	}

	return user, nil
}

func deleteDatabaseUser(ctx context.Context, client ISQLUsersService, projectName, instanceName, databaseName string) error {
	_, err := client.Delete(projectName, instanceName).Name(databaseName).Context(ctx).Do()
	if err != nil {
		return &UnleashError{Err: err, Reason: "failed to delete database user"}
	}

	return nil
}

func randomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	password := base64.URLEncoding.EncodeToString(bytes)

	return password, nil
}

func getDatabase(ctx context.Context, client ISQLDatabasesService, projectName, instanceName, databaseName string) (*admin.Database, error) {
	database, err := client.Get(projectName, instanceName, databaseName).Do()
	if err != nil {
		return database, &UnleashError{Err: err, Reason: "failed to get database"}
	}

	return database, nil
}

func deleteDatabase(ctx context.Context, client ISQLDatabasesService, projectName, instanceName, databaseName string) error {
	_, err := client.Delete(projectName, instanceName, databaseName).Do()
	if err != nil {
		return &UnleashError{Err: err, Reason: "failed to delete database"}
	}

	return nil
}

func createDatabaseUserSecret(ctx context.Context, client ctrl.Client, namespace, instanceName, instanceAddress, projectName string, database *admin.Database, user *admin.User) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      database.Name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Data: map[string][]byte{
			"POSTGRES_USER":     []byte(database.Name),
			"POSTGRES_PASSWORD": []byte(user.Password),
			"POSTGRES_DB":       []byte(database.Name),
			"POSTGRES_HOST":     []byte(instanceAddress),
		},
	}

	if err := client.Create(ctx, secret); err != nil {
		return &UnleashError{Err: err, Reason: "failed to create database user secret"}
	}

	return nil
}

func deleteDatabaseUserSecret(ctx context.Context, client ctrl.Client, namespace string, databaseName string) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      databaseName,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	if err := client.Delete(ctx, secret); err != nil {
		return &UnleashError{Err: err, Reason: "failed to delete database user secret"}
	}

	return nil
}
