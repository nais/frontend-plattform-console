package unleash

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	admin "google.golang.org/api/sqladmin/v1beta4"
)

func createDatabase(ctx context.Context, projectId string, instanceId string, databaseId string) error {
	client, err := admin.NewService(ctx)
	if err != nil {
		return err
	}

	_, err = client.Databases.Insert(projectId, instanceId, &admin.Database{
		Name: databaseId,
	}).Do()
	if err != nil {
		return err
	}

	return nil
}

func getDatabaseUser(ctx context.Context, projectId string, instanceId string, databaseId string) (*admin.User, error) {
	client, err := admin.NewService(ctx)
	if err != nil {
		return nil, err
	}

	user, err := client.Users.Get(projectId, instanceId, databaseId).Do()
	if err != nil {
		return user, err
	}

	return user, nil
}

func createDatabaseUser(ctx context.Context, projectId string, instanceId string, databaseId string) (*admin.User, error) {
	client, err := admin.NewService(ctx)
	if err != nil {
		return nil, err
	}

	password, err := randomPassword(16)
	if err != nil {
		return nil, err
	}

	user := &admin.User{
		Name:     databaseId,
		Password: password,
	}

	_, err = client.Users.Insert(projectId, instanceId, user).Do()
	if err != nil {
		return user, err
	}

	return user, nil
}

func deleteDatabaseUser(ctx context.Context, projectId string, instanceId string, databaseId string) error {
	client, err := admin.NewService(ctx)
	if err != nil {
		return err
	}

	_, err = client.Users.Delete(projectId, instanceId).Do()
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

	// Encode the byte slice in base64 format
	password := base64.URLEncoding.EncodeToString(bytes)

	return password, nil
}

func getDatabase(ctx context.Context, projectId string, instanceId string, databaseId string) (*admin.Database, error) {
	client, err := admin.NewService(ctx)
	if err != nil {
		return nil, err
	}

	database, err := client.Databases.Get(projectId, instanceId, databaseId).Do()
	if err != nil {
		return nil, err
	}

	return database, nil
}

func listDatabases(ctx context.Context, projectId string, instanceId string) ([]*admin.Database, error) {
	client, err := admin.NewService(ctx)
	if err != nil {
		return nil, err
	}

	databases, err := client.Databases.List(projectId, instanceId).Do()
	if err != nil {
		return nil, err
	}

	return databases.Items, nil
}

func deleteDatabase(ctx context.Context, projectId string, instanceId string, databaseId string) error {
	client, err := admin.NewService(ctx)
	if err != nil {
		return err
	}

	_, err = client.Databases.Delete(projectId, instanceId, databaseId).Do()
	if err != nil {
		return err
	}

	return nil
}
