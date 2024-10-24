/*
Copyright 2024 Feast Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package services

import (
	"encoding/base64"

	"gopkg.in/yaml.v3"
)

// GetServiceFeatureStoreYamlBase64 returns a base64 encoded feature_store.yaml config for the feast service
func (feast *FeastServices) GetServiceFeatureStoreYamlBase64(feastType FeastServiceType) (string, error) {
	fsYaml, err := feast.getServiceFeatureStoreYaml(feastType)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(fsYaml), nil
}

func (feast *FeastServices) getServiceFeatureStoreYaml(feastType FeastServiceType) ([]byte, error) {
	return yaml.Marshal(feast.getServiceRepoConfig(feastType))
}

func (feast *FeastServices) getServiceRepoConfig(feastType FeastServiceType) RepoConfig {
	appliedSpec := feast.FeatureStore.Status.Applied

	repoConfig := feast.getClientRepoConfig()
	if appliedSpec.Services != nil {
		if appliedSpec.Services.OfflineStore != nil && feastType == OfflineFeastType {
			// Offline server has `offline_store` section and a remote `registry`
			repoConfig.OfflineStore = OfflineStoreConfig{
				// ?? Path: LocalRegistryPath,
				Type: OfflineDaskConfigType,
			}
			repoConfig.OnlineStore = OnlineStoreConfig{}
		}
		if appliedSpec.Services.OnlineStore != nil && feastType == OnlineFeastType {
			// Online server has `online_store` section, a remote `registry` and a remote `offline_store`
			repoConfig.OnlineStore = OnlineStoreConfig{
				Type: OnlineSqliteConfigType,
				Path: LocalOnlinePath,
			}
		}
		if appliedSpec.Services.Registry != nil && feastType == RegistryFeastType {
			// Registry server has only `registry` section
			repoConfig.Registry = RegistryConfig{
				RegistryType: RegistryFileConfigType,
				Path:         LocalRegistryPath,
			}
			repoConfig.OfflineStore = OfflineStoreConfig{}
			repoConfig.OnlineStore = OnlineStoreConfig{}
		}
	}

	return repoConfig
}
