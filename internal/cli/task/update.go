package task

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docup/agentctl/internal/app/command"
	"github.com/docup/agentctl/internal/app/dto"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates the task update command.
func NewUpdateCmd(handler *command.UpdateTask) *cobra.Command {
	var (
		title                string
		goal                 string
		agent                string
		addTemplates         []string
		removeTemplates      []string
		addGuidelines        []string
		removeGuidelines     []string
		addAllowedPaths      []string
		removeAllowedPaths   []string
		addForbiddenPaths    []string
		removeForbiddenPaths []string
		addMustRead          []string
		removeMustRead       []string
		setMutations         []string
		addMutations         []string
		removeMutations      []string
	)

	cmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Configure a draft task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := dto.UpdateTaskRequest{
				TaskID:               args[0],
				AddTemplates:         addTemplates,
				RemoveTemplates:      removeTemplates,
				AddGuidelines:        addGuidelines,
				RemoveGuidelines:     removeGuidelines,
				AddAllowedPaths:      addAllowedPaths,
				RemoveAllowedPaths:   removeAllowedPaths,
				AddForbiddenPaths:    addForbiddenPaths,
				RemoveForbiddenPaths: removeForbiddenPaths,
				AddMustRead:          addMustRead,
				RemoveMustRead:       removeMustRead,
			}

			if cmd.Flags().Changed("title") {
				req.Title = &title
			}
			if cmd.Flags().Changed("goal") {
				req.Goal = &goal
			}
			if cmd.Flags().Changed("agent") {
				req.Agent = &agent
			}

			mutations, err := parseTaskMutations(setMutations, addMutations, removeMutations)
			if err != nil {
				return err
			}
			req.Mutations = mutations

			t, err := handler.Execute(req)
			if err != nil {
				return err
			}
			fmt.Printf("Updated task %s\n", t.ID)
			fmt.Printf("  Title:      %s\n", t.Title)
			fmt.Printf("  Goal:       %s\n", t.Goal)
			fmt.Printf("  Agent:      %s\n", t.Agent)
			fmt.Printf("  Templates:  %v\n", t.PromptTemplates.Builtin)
			fmt.Printf("  Guidelines: %v\n", t.Guidelines)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Set task title")
	cmd.Flags().StringVar(&goal, "goal", "", "Set engineering goal")
	cmd.Flags().StringVar(&agent, "agent", "", "Set agent (empty string clears it)")
	cmd.Flags().StringSliceVar(&addTemplates, "add-template", nil, "Add prompt template")
	cmd.Flags().StringSliceVar(&removeTemplates, "remove-template", nil, "Remove prompt template")
	cmd.Flags().StringSliceVar(&addGuidelines, "add-guideline", nil, "Add guideline")
	cmd.Flags().StringSliceVar(&removeGuidelines, "remove-guideline", nil, "Remove guideline")
	cmd.Flags().StringSliceVar(&addAllowedPaths, "add-allowed-path", nil, "Add allowed path")
	cmd.Flags().StringSliceVar(&removeAllowedPaths, "remove-allowed-path", nil, "Remove allowed path")
	cmd.Flags().StringSliceVar(&addForbiddenPaths, "add-forbidden-path", nil, "Add forbidden path")
	cmd.Flags().StringSliceVar(&removeForbiddenPaths, "remove-forbidden-path", nil, "Remove forbidden path")
	cmd.Flags().StringSliceVar(&addMustRead, "add-must-read", nil, "Add must-read file")
	cmd.Flags().StringSliceVar(&removeMustRead, "remove-must-read", nil, "Remove must-read file")
	cmd.Flags().StringSliceVar(&setMutations, "set", nil, "Set a task path: path=value")
	cmd.Flags().StringSliceVar(&addMutations, "add", nil, "Append to a list path: path=value")
	cmd.Flags().StringSliceVar(&removeMutations, "remove", nil, "Remove from a list path: path=value")
	return cmd
}

func parseTaskMutations(setOps, addOps, removeOps []string) ([]dto.TaskMutation, error) {
	var mutations []dto.TaskMutation

	appendParsed := func(kind dto.MutationKind, rawOps []string) error {
		for _, rawOp := range rawOps {
			path, rawValue, err := splitMutationArg(rawOp)
			if err != nil {
				return err
			}
			value, err := parseMutationValue(rawValue)
			if err != nil {
				return err
			}
			mutations = append(mutations, dto.TaskMutation{
				Kind:  kind,
				Path:  path,
				Value: value,
			})
		}
		return nil
	}

	if err := appendParsed(dto.MutationSet, setOps); err != nil {
		return nil, err
	}
	if err := appendParsed(dto.MutationAdd, addOps); err != nil {
		return nil, err
	}
	if err := appendParsed(dto.MutationRemove, removeOps); err != nil {
		return nil, err
	}

	return mutations, nil
}

func splitMutationArg(raw string) (string, string, error) {
	path, value, ok := strings.Cut(raw, "=")
	if !ok {
		return "", "", fmt.Errorf("mutation %q must be in path=value format", raw)
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", fmt.Errorf("mutation %q has empty path", raw)
	}
	return path, value, nil
}

func parseMutationValue(raw string) (interface{}, error) {
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err == nil {
		return value, nil
	}
	return raw, nil
}
