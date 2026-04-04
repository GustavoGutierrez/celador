package install

import (
	"fmt"

	"github.com/GustavoGutierrez/celador/internal/core/shared"
)

func CommandForManager(manager shared.PackageManager, args []string) (string, []string, error) {
	switch manager {
	case shared.PackageManagerNPM:
		return "npm", append([]string{"install"}, args...), nil
	case shared.PackageManagerPNPM:
		return "pnpm", append([]string{"install"}, args...), nil
	case shared.PackageManagerBun:
		return "bun", append([]string{"add"}, args...), nil
	default:
		return "", nil, fmt.Errorf("install is unsupported for %s", manager)
	}
}
