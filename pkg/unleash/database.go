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

func createDatabase(ctx context.Context, client *admin.Service, instance *admin.DatabaseInstance, databaseName string) (*admin.Database, error) {
	database := &admin.Database{
		Name: databaseName,
	}

	_, err := client.Databases.Insert(instance.Project, instance.Name, database).Context(ctx).Do()
	if err != nil {
		return database, err
	}

	return database, nil
}

func getDatabaseUser(ctx context.Context, client *admin.Service, instance *admin.DatabaseInstance, databaseName string) (*admin.User, error) {
	user, err := client.Users.Get(instance.Project, instance.Name, databaseName).Context(ctx).Do()
	if err != nil {
		return user, err
	}

	return user, nil
}

func createDatabaseUser(ctx context.Context, client *admin.Service, instance *admin.DatabaseInstance, databaseName string) (*admin.User, error) {
	password, err := randomPassword(16)
	if err != nil {
		return nil, err
	}

	user := &admin.User{
		Name:     databaseName,
		Password: password,
	}

	_, err = client.Users.Insert(instance.Project, instance.Name, user).Context(ctx).Do()
	if err != nil {
		return user, err
	}

	return user, nil
}

func deleteDatabaseUser(ctx context.Context, client *admin.Service, instance *admin.DatabaseInstance, databaseName string) error {
	_, err := client.Users.Delete(instance.Project, instance.Name).Name(databaseName).Context(ctx).Do()
	if err != nil {
		return err
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

func getDatabase(ctx context.Context, client *admin.Service, instance *admin.DatabaseInstance, databaseName string) (*admin.Database, error) {
	database, err := client.Databases.Get(instance.Project, instance.Name, databaseName).Do()
	if err != nil {
		return nil, err
	}

	return database, nil
}

func getDatabases(ctx context.Context, client *admin.Service, instance *admin.DatabaseInstance) ([]*admin.Database, error) {
	databases, err := client.Databases.List(instance.Project, instance.Name).Do()
	if err != nil {
		return nil, err
	}

	return databases.Items, nil
}

func deleteDatabase(ctx context.Context, client *admin.Service, instance *admin.DatabaseInstance, databaseName string) error {
	_, err := client.Databases.Delete(instance.Project, instance.Name, databaseName).Do()
	if err != nil {
		return err
	}

	return nil
}

func createDatabaseUserSecret(ctx context.Context, client ctrl.Client, namespace string, instance *admin.DatabaseInstance, database *admin.Database, user *admin.User) error {
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
			"POSTGRES_HOST":     []byte(instance.IpAddresses[0].IpAddress),
		},
	}

	err := client.Create(ctx, secret)
	return err
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
	return client.Delete(ctx, secret)
}
