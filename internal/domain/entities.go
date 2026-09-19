package domain

// Entities returns all database model instances for auto-migration
func Entities() []interface{} {
	return []interface{}{
		&User{},
	}
}
