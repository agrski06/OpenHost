package system

import "context"

func (m *Manager) EnsureUser(ctx context.Context, user string) error {
	if user == "" {
		return nil
	}

	m.Logger.Info("ensuring system user", "user", user)
	return m.Executor.Run(ctx, "sh", "-c", "id -u \"$1\" >/dev/null 2>&1 || useradd -m -s /bin/bash \"$1\"", "sh", user)
}

func (m *Manager) ChownR(ctx context.Context, path string, user string) error {
	if path == "" || user == "" {
		return nil
	}
	m.Logger.Info("updating ownership", "path", path, "user", user)
	return m.Executor.Run(ctx, "chown", "-R", user+":"+user, path)
}
