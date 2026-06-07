package command

import (
	"fmt"
	"strings"
)

type Command interface {
	Name() string
	Aliases() []string
	Description() string
	Execute(args []string) (string, error)
}

type Registry struct {
	commands map[string]Command
}

func NewRegistry() *Registry {
	r := &Registry{
		commands: make(map[string]Command),
	}

	r.Register(&ConnectCommand{})
	r.Register(&InitCommand{})
	r.Register(&UndoCommand{})
	r.Register(&RedoCommand{})
	r.Register(&HelpCommand{})
	r.Register(&ExitCommand{})
	r.Register(&ModelCommand{})
	r.Register(&SessionCommand{})
	r.Register(&CompactCommand{})
	r.Register(&ExportCommand{})
	r.Register(&ShareCommand{})
	r.Register(&ClearCommand{})
	r.Register(&ModeCommand{})
	r.Register(&ModelsCommand{})
	r.Register(&PermissionCommand{})

	return r
}

func (r *Registry) Register(cmd Command) {
	r.commands[cmd.Name()] = cmd
	for _, alias := range cmd.Aliases() {
		r.commands[alias] = cmd
	}
}

func (r *Registry) Execute(input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmdName := strings.TrimPrefix(strings.ToLower(parts[0]), "/")
	args := parts[1:]

	cmd, ok := r.commands[cmdName]
	if !ok {
		return "", fmt.Errorf("unknown command: %s. Type /help for available commands.", cmdName)
	}

	return cmd.Execute(args)
}

func (r *Registry) List() []Command {
	seen := make(map[string]bool)
	cmds := make([]Command, 0)

	for _, cmd := range r.commands {
		if !seen[cmd.Name()] {
			seen[cmd.Name()] = true
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}
