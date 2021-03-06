package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var (
	changeLogURL            = "https://github.com/urfave/cli/blob/master/docs/CHANGELOG.md"
	appActionDeprecationURL = fmt.Sprintf("%s#deprecated-cli-app-action-signature", changeLogURL)
	contactSysadmin         = "This is an error in the application.  Please contact the distributor of this application if this is not you."
	errInvalidActionType    = Exit("ERROR invalid Action type. "+
		fmt.Sprintf("Must be `func(*Context`)` or `func(*Context) error).  %s", contactSysadmin)+
		fmt.Sprintf("See %s", appActionDeprecationURL), 2)
)

// App is the main structure of a cli application. It is recommended that
// an app be created with the cli.NewApp() function
type App struct {
	// The name of the program. Defaults to path.Base(os.Args[0])
	Name string
	// Full name of command for help, defaults to Name
	HelpName string
	// Description of the program.
	Usage string
	// Text to override the USAGE section of help
	UsageText string
	// Description of the program argument format.
	ArgsUsage string
	// Version of the program
	Version string
	// Description of the program
	Description string
	// List of commands to execute
	Commands []*Command
	// List of flags to parse
	Flags []Flag
	// Boolean to enable bash completion commands
	EnableBashCompletion bool
	// Boolean to hide built-in help command and help flag
	HideHelp bool
	// Boolean to hide built-in help command but keep help flag.
	// Ignored if HideHelp is true.
	HideHelpCommand bool
	// Boolean to hide built-in version flag and the VERSION section of help
	HideVersion bool
	// categories contains the categorized commands and is populated on app startup
	categories CommandCategories
	// An action to execute when the shell completion flag is set
	BashComplete BashCompleteFunc
	// An action to execute before any subcommands are run, but after the context is ready
	// If a non-nil error is returned, no subcommands are run
	Before BeforeFunc
	// An action to execute after any subcommands are run, but after the subcommand has finished
	// It is run even if Action() panics
	After AfterFunc
	// The action to execute when no subcommands are specified
	Action ActionFunc
	// Execute this function if the proper command cannot be found
	CommandNotFound CommandNotFoundFunc
	// Execute this function if an usage error occurs
	OnUsageError OnUsageErrorFunc
	// Compilation date
	Compiled time.Time
	// List of all authors who contributed
	Authors []*Author
	// Copyright of the binary if any
	Copyright string
	// Writer writer to write output to
	Writer io.Writer
	// ErrWriter writes error output
	ErrWriter io.Writer
	// ExitErrHandler processes any error encountered while running an App before
	// it is returned to the caller. If no function is provided, HandleExitCoder
	// is used as the default behavior.
	ExitErrHandler ExitErrHandlerFunc
	// Other custom info
	Metadata map[string]interface{}
	// Carries a function which returns app specific info.
	ExtraInfo func() map[string]string
	// CustomAppHelpTemplate the text template for app help topic.
	// cli.go uses text/template to render templates. You can
	// render custom help text by setting this variable.
	CustomAppHelpTemplate string
	// Boolean to enable short-option handling so user can combine several
	// single-character bool arguments into one
	// i.e. foobar -o -v -> foobar -ov
	UseShortOptionHandling bool

	didSetup    bool
	aliases     []string
	category    string
	isSubmodule bool
}

// CompileTime tries to find out when this binary was compiled.
// Returns the current time if it fails to find it.
func CompileTime() time.Time {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

// NewApp creates a new cli Application with some reasonable defaults for Name,
// Usage, Version and Action.
func NewApp() *App {
	return &App{
		Name:         filepath.Base(os.Args[0]),
		HelpName:     filepath.Base(os.Args[0]),
		Usage:        "A new cli application",
		UsageText:    "",
		BashComplete: DefaultAppComplete,
		Action:       helpCommand.Action,
		Compiled:     CompileTime(),
		Writer:       os.Stdout,
		ErrWriter:    os.Stderr,
	}
}

// Setup runs initialization code to ensure all data structures are ready for
// `Run` or inspection prior to `Run`.  It is internally called by `Run`, but
// will return early if setup has already happened.
func (a *App) Setup() {
	if a.didSetup {
		return
	}

	a.didSetup = true

	if a.Name == "" {
		a.Name = filepath.Base(os.Args[0])
	}

	if a.HelpName == "" {
		a.HelpName = filepath.Base(os.Args[0])
	}

	if a.Usage == "" {
		a.Usage = "A new cli application"
	}

	if a.Version == "" {
		a.HideVersion = true
	}

	if a.BashComplete == nil {
		a.BashComplete = DefaultAppComplete
	}

	if a.Action == nil {
		a.Action = helpCommand.Action
	}

	if a.Compiled == (time.Time{}) {
		a.Compiled = CompileTime()
	}

	if a.Writer == nil {
		a.Writer = os.Stdout
	}

	if a.ErrWriter == nil {
		a.ErrWriter = os.Stderr
	}

	var newCommands []*Command

	for _, c := range a.Commands {
		if c.HelpName == "" {
			c.HelpName = fmt.Sprintf("%s [options] %s", a.HelpName, c.Name)
		}
		newCommands = append(newCommands, c)
	}
	a.Commands = newCommands

	if a.Command(helpCommand.Name) == nil && !a.HideHelp {
		if !a.HideHelpCommand {
			a.appendCommand(helpCommand)
		}

		if HelpFlag != nil {
			a.appendFlag(HelpFlag)
		}
	}

	if !a.HideVersion {
		a.appendFlag(VersionFlag)
	}

	a.categories = newCommandCategories()
	for _, command := range a.Commands {
		a.categories.AddCommand(command.Category, command)
	}
	sort.Sort(a.categories.(*commandCategories))

	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}
}

func (a *App) newFlagSet() (*flag.FlagSet, error) {
	return flagSet(a.Name, a.Flags)
}

func (a *App) useShortOptionHandling() bool {
	return a.UseShortOptionHandling
}

// Run is the entry point to the cli app. Parses the arguments slice and routes
// to the proper flag/args combination
func (a *App) Run(arguments []string) (err error) {
	return a.RunContext(context.Background(), arguments)
}

// RunContext is like Run except it takes a Context that will be
// passed to its commands and sub-commands. Through this, you can
// propagate timeouts and cancellation requests
func (a *App) RunContext(ctx context.Context, arguments []string) (err error) {
	a.Setup()

	// handle the completion flag separately from the flagset since
	// completion could be attempted after a flag, but before its value was put
	// on the command line. this causes the flagset to interpret the completion
	// flag name as the value of the flag before it which is undesirable
	// note that we can only do this because the shell autocomplete function
	// always appends the completion flag at the end of the command
	shellComplete, arguments := checkShellCompleteFlag(a, arguments)

	set, err := a.newFlagSet()
	if err != nil {
		return err
	}

	err = parseIter(set, a, arguments[1:], shellComplete)
	nerr := normalizeFlags(a.Flags, set)
	newContext := NewContext(a, set, &Context{Context: ctx})
	if nerr != nil {
		_, _ = fmt.Fprintln(a.Writer, nerr)
		_ = ShowAppHelp(newContext)
		return nerr
	}
	newContext.shellComplete = shellComplete

	if checkCompletions(newContext) {
		return nil
	}

	if err != nil {
		if a.OnUsageError != nil {
			err := a.OnUsageError(newContext, err, false)
			a.handleExitCoder(newContext, err)
			return err
		}
		_, _ = fmt.Fprintf(a.Writer, "%s %s\n\n", "Incorrect Usage.", err.Error())
		_ = ShowAppHelp(newContext)
		return err
	}

	if !a.HideHelp && checkHelp(newContext) {
		_ = ShowAppHelp(newContext)
		return nil
	}

	if !a.HideVersion && checkVersion(newContext) {
		ShowVersion(newContext)
		return nil
	}

	cerr := checkRequiredFlags(a.Flags, newContext)
	if cerr != nil {
		_ = ShowAppHelp(newContext)
		return cerr
	}

	if a.After != nil {
		defer func() {
			if afterErr := a.After(newContext); afterErr != nil {
				if err != nil {
					err = newMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if a.Before != nil {
		beforeErr := a.Before(newContext)
		if beforeErr != nil {
			a.handleExitCoder(newContext, beforeErr)
			err = beforeErr
			return err
		}
	}

	args := newContext.Args()
	if args.Present() {
		name := args.First()
		c := a.Command(name)
		if c != nil {
			return c.Run(newContext)
		}
	}

	if a.Action == nil {
		a.Action = helpCommand.Action
	}

	// Run default Action
	err = a.Action(newContext)

	a.handleExitCoder(newContext, err)
	return err
}

// RunAndExitOnError calls .Run() and exits non-zero if an error was returned
//
// Deprecated: instead you should return an error that fulfills cli.ExitCoder
// to cli.App.Run. This will cause the application to exit with the given eror
// code in the cli.ExitCoder
func (a *App) RunAndExitOnError() {
	if err := a.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(a.ErrWriter, err)
		OsExiter(1)
	}
}

// IsSubmodule returns if the app instance is a submodule
func (a *App) IsSubmodule() bool {
	return a.isSubmodule
}

// RunAsSubcommand invokes the subcommand given the context, parses ctx.Args() to
// generate command-specific flags
func (a *App) RunAsSubcommand(ctx *Context, parentCommand *Command) (err error) {
	// Setup also handles HideHelp and HideHelpCommand
	a.Setup()

	var newCmds []*Command
	for _, c := range a.Commands {
		if c.HelpName == "" {
			c.HelpName = fmt.Sprintf("%s [options] %s", a.HelpName, c.Name)
		}
		newCmds = append(newCmds, c)
	}
	a.Commands = newCmds

	set, err := a.newFlagSet()
	if err != nil {
		return err
	}

	err = parseIter(set, a, ctx.Args().Tail(), ctx.shellComplete)
	nerr := normalizeFlags(a.Flags, set)
	newContext := NewContext(a, set, ctx)
	a.category = parentCommand.Category
	a.aliases = parentCommand.Aliases
	if nerr != nil {
		_, _ = fmt.Fprintln(a.Writer, nerr)
		_, _ = fmt.Fprintln(a.Writer)
		if len(a.Commands) > 0 {
			_ = ShowSubcommandHelp(newContext)
		} else {
			_ = ShowCommandHelp(ctx, newContext.Args().First())
		}
		return nerr
	}

	if checkCompletions(newContext) {
		return nil
	}

	if err != nil {
		if a.OnUsageError != nil {
			err = a.OnUsageError(newContext, err, true)
			a.handleExitCoder(newContext, err)
			return err
		}
		_, _ = fmt.Fprintf(a.Writer, "%s %s\n\n", "Incorrect Usage.", err.Error())
		_ = ShowSubcommandHelp(newContext)
		return err
	}

	if len(a.Commands) > 0 {
		if checkSubcommandHelp(newContext) {
			return nil
		}
	} else {
		if checkCommandHelp(ctx, newContext.Args().First()) {
			return nil
		}
	}

	cerr := checkRequiredFlags(a.Flags, newContext)
	if cerr != nil {
		_ = ShowSubcommandHelp(newContext)
		return cerr
	}

	if a.After != nil {
		defer func() {
			afterErr := a.After(newContext)
			if afterErr != nil {
				a.handleExitCoder(newContext, err)
				if err != nil {
					err = newMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if a.Before != nil {
		beforeErr := a.Before(newContext)
		if beforeErr != nil {
			a.handleExitCoder(newContext, beforeErr)
			err = beforeErr
			return err
		}
	}

	args := newContext.Args()
	if args.Present() {
		name := args.First()
		c := a.Command(name)
		if c != nil {
			return c.Run(newContext)
		}
	}

	// Run default Action
	err = a.Action(newContext)

	a.handleExitCoder(newContext, err)
	return err
}

// Command returns the named command on App. Returns nil if the command does not exist
func (a *App) Command(name string) *Command {
	for _, c := range a.Commands {
		if c.HasName(name) {
			return c
		}
	}

	return nil
}

// VisibleCategories returns a slice of categories and commands that are
// Hidden=false
func (a *App) VisibleCategories() []CommandCategory {
	var ret []CommandCategory
	for _, category := range a.categories.Categories() {
		if visible := func() CommandCategory {
			if len(category.VisibleCommands()) > 0 {
				return category
			}
			return nil
		}(); visible != nil {
			ret = append(ret, visible)
		}
	}
	return ret
}

// Aliases returns the aliases of the app
func (a *App) Aliases() []string {
	return a.aliases
}

// Category returns the category of the app
func (a *App) Category() string {
	return a.category
}

// VisibleCommands returns a slice of the Commands with Hidden=false
func (a *App) VisibleCommands() []*Command {
	var ret []*Command
	for _, command := range a.Commands {
		if !command.Hidden {
			ret = append(ret, command)
		}
	}
	return ret
}

// VisibleFlags returns a slice of the Flags with Hidden=false
func (a *App) VisibleFlags() []Flag {
	return visibleFlags(a.Flags)
}

func (a *App) appendFlag(fl Flag) {
	if !hasFlag(a.Flags, fl) {
		a.Flags = append(a.Flags, fl)
	}
}

func (a *App) appendCommand(c *Command) {
	if !hasCommand(a.Commands, c) {
		a.Commands = append(a.Commands, c)
	}
}

func (a *App) handleExitCoder(context *Context, err error) {
	if a.ExitErrHandler != nil {
		a.ExitErrHandler(context, err)
	} else {
		HandleExitCoder(err)
	}
}

// Author represents someone who has contributed to a cli project.
type Author struct {
	Name  string // The Authors name
	Email string // The Authors email
}

// String makes Author comply to the Stringer interface, to allow an easy print in the templating process
func (a *Author) String() string {
	e := ""
	if a.Email != "" {
		e = " <" + a.Email + ">"
	}

	return fmt.Sprintf("%v%v", a.Name, e)
}

// HandleAction attempts to figure out which Action signature was used.  If
// it's an ActionFunc or a func with the legacy signature for Action, the func
// is run!
func HandleAction(action interface{}, context *Context) (err error) {
	switch a := action.(type) {
	case ActionFunc:
		return a(context)
	case func(*Context) error:
		return a(context)
	case func(*Context): // deprecated function signature
		a(context)
		return nil
	}

	return errInvalidActionType
}
