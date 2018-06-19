package graphqlbackend

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/internal/db"
	"github.com/sourcegraph/sourcegraph/pkg/actor"
	"github.com/sourcegraph/sourcegraph/pkg/conf"
)

func (schemaResolver) Configuration() *configurationCascadeResolver {
	return &configurationCascadeResolver{}
}

// configurationCasecadeResolver resolves settings from multiple sources.
// When there is overlap between configuration values they will be merged in the
// following cascading order (first is lowest-priority):
// 1. Global site server configuration
// 2. Global site settings
// 3. Organization settings
// 4. Current user settings
type configurationCascadeResolver struct{}

func (r *configurationCascadeResolver) Defaults() *configurationResolver {
	return &configurationResolver{
		contents: `// This is the default configuration. Override it to configure settings.
{
  /* default configuration is empty */
}`,
	}
}

var mockConfigurationCascadeSubjects func() ([]*configurationSubject, error)

func (r *configurationCascadeResolver) Subjects(ctx context.Context) ([]*configurationSubject, error) {
	if mockConfigurationCascadeSubjects != nil {
		return mockConfigurationCascadeSubjects()
	}

	subjects := []*configurationSubject{
		{site: singletonSiteResolver},
	}

	// Apply global site settings
	subjects = append(subjects, &configurationSubject{})

	if actor := actor.FromContext(ctx); actor.IsAuthenticated() {

		user, err := currentUser(ctx)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, nil // actor might be invalid or refer to since-deleted user
		}

		orgs, err := db.Orgs.GetByUserID(ctx, user.user.ID)
		if err != nil {
			return nil, err
		}
		// Stable-sort the orgs so that the priority of their configs is stable.
		sort.Slice(orgs, func(i, j int) bool {
			return orgs[i].ID < orgs[j].ID
		})
		// Apply the user's orgs' configuration.
		for _, org := range orgs {
			subjects = append(subjects, &configurationSubject{org: &orgResolver{org}})
		}

		// Apply the user's own configuration last (it has highest priority).
		subjects = append(subjects, &configurationSubject{user: user})
	}

	return subjects, nil
}

func (r *configurationCascadeResolver) Merged(ctx context.Context) (*configurationResolver, error) {
	configs := []string{r.Defaults().Contents()}
	subjects, err := r.Subjects(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range subjects {
		settings, err := s.LatestSettings(ctx)
		if err != nil {
			return nil, err
		}
		if settings != nil {
			configs = append(configs, settings.settings.Contents)
		}
	}

	var messages []string
	merged, err := mergeConfigs(configs)
	if err != nil {
		messages = append(messages, err.Error())
	}
	return &configurationResolver{contents: string(merged), messages: messages}, nil
}

// deeplyMergedConfigFields contains the names of top-level configuration fields whose values should
// be merged if they appear in multiple cascading configurations.
//
// For example, suppose org config is {"a":[1]} and user config is {"a":[2]}. If "a" is NOT a deeply
// merged field, the merged config would be {"a":[2]}. If "a" IS a deeply merged field, then the
// merged config would be {"a":[1,2].}
var deeplyMergedConfigFields = map[string]struct{}{
	"search.scopes":           {},
	"search.savedQueries":     {},
	"search.repositoryGroups": {},
	"motd": {},
}

// mergeConfigs merges the specified JSON configs together to produce a single JSON config. The merge
// algorithm is currently rudimentary but eventually it will be similar to VS Code's. The only "smart"
// merging behavior is that described in the documentation for deeplyMergedConfigFields.
func mergeConfigs(jsonConfigStrings []string) ([]byte, error) {
	var errs []error
	merged := map[string]interface{}{}
	for _, s := range jsonConfigStrings {
		var config map[string]interface{}
		if err := conf.UnmarshalJSON(s, &config); err != nil {
			errs = append(errs, err)
			continue
		}
		for name, value := range config {
			// See if we should deeply merge this field.
			if _, ok := deeplyMergedConfigFields[name]; ok {
				if mv, ok := merged[name].([]interface{}); merged[name] == nil || ok {
					if cv, ok := value.([]interface{}); merged[name] != nil || (value != nil && ok) {
						merged[name] = append(mv, cv...)
						continue
					}
				} else if mv, ok := merged[name].(map[string]interface{}); merged[name] == nil || ok {
					if cv, ok := value.(map[string]interface{}); merged[name] != nil || (value != nil && ok) {
						for key, value := range cv {
							mv[key] = value
						}
						merged[name] = mv
						continue
					}
				}
			}

			// Otherwise clobber any existing value.
			merged[name] = value
		}
	}
	out, err := json.Marshal(merged)
	if err != nil {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return out, nil
	}
	return out, fmt.Errorf("errors merging configurations: %q", errs)
}