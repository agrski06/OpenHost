package system

import "fmt"

// CreateUser creates a system user with a home directory and bash shell.
func CreateUser(name string) error {
	if err := run("useradd", "-m", "-s", "/bin/bash", name); err != nil {
		return fmt.Errorf("create user %q: %w", name, err)
	}
	return nil
}

// SetOwnership recursively sets ownership of a path to the given user.
func SetOwnership(user, path string) error {
	ownership := fmt.Sprintf("%s:%s", user, user)
	if err := run("chown", "-R", ownership, path); err != nil {
		return fmt.Errorf("set ownership %s on %q: %w", ownership, path, err)
	}
	return nil
}
