package command

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/docup/agentctl/internal/app/dto"
	"github.com/docup/agentctl/internal/core/task"
	"gopkg.in/yaml.v3"
)

var configurableTaskRoots = map[string]struct{}{
	"title":            {},
	"goal":             {},
	"agent":            {},
	"prompt_templates": {},
	"scope":            {},
	"guidelines":       {},
	"context":          {},
	"constraints":      {},
	"interaction":      {},
	"runtime":          {},
	"validation":       {},
}

func applyTaskUpdate(t *task.Task, req dto.UpdateTaskRequest) error {
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Goal != nil {
		t.Goal = *req.Goal
	}
	if req.Agent != nil {
		t.Agent = *req.Agent
	}

	applyStringSliceAdd(&t.PromptTemplates.Builtin, req.AddTemplates)
	applyStringSliceRemove(&t.PromptTemplates.Builtin, req.RemoveTemplates)
	applyStringSliceAdd(&t.Guidelines, req.AddGuidelines)
	applyStringSliceRemove(&t.Guidelines, req.RemoveGuidelines)
	applyStringSliceAdd(&t.Scope.AllowedPaths, req.AddAllowedPaths)
	applyStringSliceRemove(&t.Scope.AllowedPaths, req.RemoveAllowedPaths)
	applyStringSliceAdd(&t.Scope.ForbiddenPaths, req.AddForbiddenPaths)
	applyStringSliceRemove(&t.Scope.ForbiddenPaths, req.RemoveForbiddenPaths)
	applyStringSliceAdd(&t.Scope.MustRead, req.AddMustRead)
	applyStringSliceRemove(&t.Scope.MustRead, req.RemoveMustRead)

	if err := validateExplicitMutationOverlap(req); err != nil {
		return err
	}

	for _, mutation := range req.Mutations {
		if err := applyGenericMutation(t, mutation); err != nil {
			return err
		}
	}

	return nil
}

func validateExplicitMutationOverlap(req dto.UpdateTaskRequest) error {
	explicitPaths := map[string]bool{}
	if req.Title != nil {
		explicitPaths["title"] = true
	}
	if req.Goal != nil {
		explicitPaths["goal"] = true
	}
	if req.Agent != nil {
		explicitPaths["agent"] = true
	}
	if len(req.AddTemplates) > 0 || len(req.RemoveTemplates) > 0 {
		explicitPaths["prompt_templates.builtin"] = true
	}
	if len(req.AddGuidelines) > 0 || len(req.RemoveGuidelines) > 0 {
		explicitPaths["guidelines"] = true
	}
	if len(req.AddAllowedPaths) > 0 || len(req.RemoveAllowedPaths) > 0 {
		explicitPaths["scope.allowed_paths"] = true
	}
	if len(req.AddForbiddenPaths) > 0 || len(req.RemoveForbiddenPaths) > 0 {
		explicitPaths["scope.forbidden_paths"] = true
	}
	if len(req.AddMustRead) > 0 || len(req.RemoveMustRead) > 0 {
		explicitPaths["scope.must_read"] = true
	}

	for _, mutation := range req.Mutations {
		path := canonicalTaskPath(mutation.Path)
		if explicitPaths[path] {
			return fmt.Errorf("path %q is already modified by an explicit update flag", path)
		}
	}

	return nil
}

func applyStringSliceAdd(dst *[]string, values []string) {
	for _, value := range values {
		if !containsExact(*dst, value) {
			*dst = append(*dst, value)
		}
	}
}

func applyStringSliceRemove(dst *[]string, values []string) {
	if len(values) == 0 || len(*dst) == 0 {
		return
	}

	filtered := (*dst)[:0]
	for _, existing := range *dst {
		if !containsExact(values, existing) {
			filtered = append(filtered, existing)
		}
	}
	*dst = filtered
}

func containsExact(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func applyGenericMutation(t *task.Task, mutation dto.TaskMutation) error {
	path := canonicalTaskPath(mutation.Path)
	if path == "" {
		return fmt.Errorf("mutation path cannot be empty")
	}
	if err := validateConfigurablePath(path); err != nil {
		return err
	}

	target, err := resolveTaskPath(reflect.ValueOf(t), strings.Split(path, "."))
	if err != nil {
		return err
	}

	switch mutation.Kind {
	case dto.MutationSet:
		return setValue(target, mutation.Value)
	case dto.MutationAdd:
		return addSliceValue(target, mutation.Value)
	case dto.MutationRemove:
		return removeSliceValue(target, mutation.Value)
	default:
		return fmt.Errorf("unsupported mutation kind %q", mutation.Kind)
	}
}

func validateConfigurablePath(path string) error {
	root, _, _ := strings.Cut(path, ".")
	if _, ok := configurableTaskRoots[root]; !ok {
		return fmt.Errorf("path %q is not configurable", path)
	}
	return nil
}

func canonicalTaskPath(path string) string {
	return strings.TrimSpace(path)
}

func resolveTaskPath(value reflect.Value, segments []string) (reflect.Value, error) {
	current := value
	if current.Kind() == reflect.Pointer {
		if current.IsNil() {
			return reflect.Value{}, fmt.Errorf("nil task value")
		}
		current = current.Elem()
	}

	for _, segment := range segments {
		if current.Kind() == reflect.Pointer {
			if current.IsNil() {
				current.Set(reflect.New(current.Type().Elem()))
			}
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("path %q does not resolve to a struct field", strings.Join(segments, "."))
		}

		field, ok := findFieldByYAMLTag(current, segment)
		if !ok {
			return reflect.Value{}, fmt.Errorf("unknown task path %q", strings.Join(segments, "."))
		}
		current = field
	}

	return current, nil
}

func findFieldByYAMLTag(structValue reflect.Value, segment string) (reflect.Value, bool) {
	structType := structValue.Type()
	for i := 0; i < structValue.NumField(); i++ {
		fieldType := structType.Field(i)
		tag := fieldType.Tag.Get("yaml")
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		if name == segment {
			return structValue.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func setValue(target reflect.Value, raw interface{}) error {
	if target.Kind() == reflect.Pointer {
		if raw == nil {
			target.Set(reflect.Zero(target.Type()))
			return nil
		}
		decoded, err := decodeValue(raw, target.Type().Elem())
		if err != nil {
			return err
		}
		ptr := reflect.New(target.Type().Elem())
		ptr.Elem().Set(decoded)
		target.Set(ptr)
		return nil
	}

	decoded, err := decodeValue(raw, target.Type())
	if err != nil {
		return err
	}
	target.Set(decoded)
	return nil
}

func addSliceValue(target reflect.Value, raw interface{}) error {
	if target.Kind() != reflect.Slice {
		return fmt.Errorf("path %q does not point to a list field", target.Type())
	}

	decoded, err := decodeValue(raw, target.Type().Elem())
	if err != nil {
		return err
	}

	for i := 0; i < target.Len(); i++ {
		if reflect.DeepEqual(target.Index(i).Interface(), decoded.Interface()) {
			return nil
		}
	}

	target.Set(reflect.Append(target, decoded))
	return nil
}

func removeSliceValue(target reflect.Value, raw interface{}) error {
	if target.Kind() != reflect.Slice {
		return fmt.Errorf("path %q does not point to a list field", target.Type())
	}

	decoded, err := decodeValue(raw, target.Type().Elem())
	if err != nil {
		return err
	}

	filtered := reflect.MakeSlice(target.Type(), 0, target.Len())
	for i := 0; i < target.Len(); i++ {
		if reflect.DeepEqual(target.Index(i).Interface(), decoded.Interface()) {
			continue
		}
		filtered = reflect.Append(filtered, target.Index(i))
	}

	target.Set(filtered)
	return nil
}

func decodeValue(raw interface{}, targetType reflect.Type) (reflect.Value, error) {
	payload, err := yaml.Marshal(raw)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("encoding value for %s: %w", targetType, err)
	}

	decoded := reflect.New(targetType)
	if err := yaml.Unmarshal(payload, decoded.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("decoding value for %s: %w", targetType, err)
	}
	return decoded.Elem(), nil
}
