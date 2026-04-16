package environments

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	mesheryctlflags "github.com/meshery/meshery/mesheryctl/internal/cli/pkg/flags"
	"github.com/meshery/meshery/mesheryctl/pkg/utils"
)

func TestCreateEnvironment(t *testing.T) {
	// Get current directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Not able to get current working directory")
	}
	currDir := filepath.Dir(filename)

	// Test scenarios for environment creation
	tests := []utils.MesheryCommandTest{
		{
			Name:             "Create environment without arguments",
			Args:             []string{"create"},
			URL:              "/api/environments",
			HttpMethod:       "POST",
			Fixture:          "",
			ExpectedResponse: "",
			ExpectError:      true,
			IsOutputGolden:   false,
			ExpectedError:    utils.ErrFlagsInvalid(fmt.Errorf("Invalid value for --orgId '', Invalid value for --name '', Invalid value for --description ''")),
		},
		{
			Name:             "Create environment with invalid organization ID",
			Args:             []string{"create", "--name", testConstants["environmentName"], "--description", "integration test", "--orgId", "invalid-org-id"},
			URL:              "/api/environments",
			HttpMethod:       "POST",
			Fixture:          "",
			ExpectedResponse: "",
			ExpectError:      true,
			IsOutputGolden:   false,
			ExpectedError:    utils.ErrFlagsInvalid(fmt.Errorf("Invalid value for --orgid 'invalid-org-id': must be a valid UUID")),
		},
		{
			Name:             "Create environment with empty required flag values",
			Args:             []string{"create", "--name", "", "--description", "", "--orgId", ""},
			URL:              "/api/environments",
			HttpMethod:       "POST",
			Fixture:          "",
			ExpectedResponse: "",
			ExpectError:      true,
			IsOutputGolden:   false,
			ExpectedError:    utils.ErrFlagsInvalid(fmt.Errorf("Invalid value for --orgId '', Invalid value for --name '', Invalid value for --description ''")),
		},
		{
			Name:             "Create environment successfully",
			Args:             []string{"create", "--name", testConstants["environmentName"], "--description", "integration test", "--orgId", testConstants["orgId"]},
			URL:              "/api/environments",
			HttpMethod:       "POST",
			HttpStatusCode:   200,
			Fixture:          "create.environment.response.golden",
			ExpectedResponse: "create.environment.success.golden",
			ExpectError:      false,
		},
	}

	mesheryctlflags.InitValidators(EnvironmentCmd)
	utils.InvokeMesheryctlTestCommand(t, update, EnvironmentCmd, tests, currDir, "environments")
}
