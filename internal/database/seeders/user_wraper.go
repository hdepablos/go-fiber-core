package seeders

func createUsersWrapper() error {
	return CreateUserSeeder(defaultConfigPath)
}