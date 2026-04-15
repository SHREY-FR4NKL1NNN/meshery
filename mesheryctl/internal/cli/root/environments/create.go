// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package environments

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/meshery/meshery/mesheryctl/internal/cli/pkg/api"
	"github.com/meshery/meshery/mesheryctl/pkg/utils"
	mErrors "github.com/meshery/meshkit/errors"
	"github.com/meshery/schemas/models/v1beta1/environment"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	createEnvironmentPayload environment.EnvironmentPayload
	createEnvironmentOrgId   string
)

var createEnvironmentCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new environment",
	Long: `Create a new environment by providing the name and description of the environment
Find more information at: https://docs.meshery.io/reference/mesheryctl/environment/create`,
	Example: `
// Create a new environment
mesheryctl environment create --orgId [orgId] --name [name] --description [description]
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		const errMsg = "[ Organization ID | Name | Description ] aren't specified\n\nUsage: mesheryctl environment create --orgId [orgId] --name [name] --description [description]\nRun 'mesheryctl environment create --help' to see detailed help message"

		if createEnvironmentOrgId == "" || createEnvironmentPayload.Name == "" || createEnvironmentPayload.Description == "" {
			return utils.ErrInvalidArgument(errors.New(errMsg))
		}

		if !utils.IsUUID(createEnvironmentOrgId) {
			return utils.ErrInvalidUUID(fmt.Errorf("invalid Organization ID: %s", createEnvironmentOrgId))
		}

		createEnvironmentPayload.OrgId = uuid.MustParse(createEnvironmentOrgId)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		payloadBytes, err := json.Marshal(&createEnvironmentPayload)
		if err != nil {
			return err
		}
		_, err = api.Add(environmentApiPath, bytes.NewBuffer(payloadBytes), nil)
		if err != nil {
			if meshkitErr, ok := err.(*mErrors.Error); ok {
				if meshkitErr.Code == utils.ErrFailReqStatusCode {
					return errCreateEnvironment(createEnvironmentPayload.Name, createEnvironmentOrgId)
				}
			}
			return err
		}

		utils.Log.Infof("Environment named %s created in organization id %s", createEnvironmentPayload.Name, createEnvironmentOrgId)
		return nil
	},
}

func init() {
	createEnvironmentCmd.Flags().StringVarP(&createEnvironmentOrgId, "orgId", "o", "", "Organization ID")
	createEnvironmentCmd.Flags().StringVarP(&createEnvironmentPayload.Name, "name", "n", "", "Name of the environment")
	createEnvironmentCmd.Flags().StringVarP(&createEnvironmentPayload.Description, "description", "d", "", "Description of the environment")
}
