package unleash

import "context"

type Unleash struct {
	ProjectId    string
	Instance     string
	DatabaseName string
	DatabaseUser string
}

func GetInstances(ctx context.Context, projectId string, instanceId string) ([]Unleash, error) {
	databases, err := listDatabases(ctx, projectId, instanceId)
	if err != nil {
		return nil, err
	}

	var instances []Unleash

	for _, database := range databases {
		if database.Name == "postgres" {
			continue
		}

		instances = append(instances, Unleash{
			ProjectId:    projectId,
			Instance:     instanceId,
			DatabaseName: database.Name,
		})
	}

	return instances, nil
}

func GetInstance(ctx context.Context, projectId string, instanceId string, databaseId string) (Unleash, error) {
	database, err := getDatabase(ctx, projectId, instanceId, databaseId)
	if err != nil {
		return Unleash{}, err
	}

	databaseUser, err := getDatabaseUser(ctx, projectId, instanceId, databaseId)
	if err != nil {
		return Unleash{}, err
	}

	return Unleash{
		ProjectId:    projectId,
		Instance:     instanceId,
		DatabaseName: database.Name,
		DatabaseUser: databaseUser.Name,
	}, nil
}

func CreateInstance(ctx context.Context, projectId string, instanceId string, databaseId string) (Unleash, error) {
	err := createDatabase(ctx, projectId, instanceId, databaseId)
	if err != nil {
		return Unleash{}, err
	}

	databaseUser, err := createDatabaseUser(ctx, projectId, instanceId, databaseId)
	if err != nil {
		return Unleash{}, err
	}

	// TODO: create kubernetes secret
	// TODO: create unleash instance

	return Unleash{
		ProjectId:    projectId,
		Instance:     instanceId,
		DatabaseName: databaseId,
		DatabaseUser: databaseUser.Name,
	}, nil
}

func DeleteInstance(ctx context.Context, projectId string, instanceId string, databaseId string) error {
	// TODO: delete unleash instance
	// TODO: delete kubernetes secret

	err := deleteDatabaseUser(ctx, projectId, instanceId, databaseId)
	if err != nil {
		return err
	}

	err = deleteDatabase(ctx, projectId, instanceId, databaseId)
	if err != nil {
		return err
	}

	return nil
}
