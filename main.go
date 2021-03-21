package main

import (
	_ "bufio"
	"fmt"
	"os"
	"os/exec"
	_ "os/user"
	"syscall"
	"os/signal"
	"strings"
	"io"
	lfs "hilbish/golibs/fs"
	cmds "hilbish/golibs/commander"

	"github.com/akamensky/argparse"
	"github.com/bobappleyard/readline"
	"github.com/yuin/gopher-lua"
	"layeh.com/gopher-luar"
)

const version = "0.0.12"
var l *lua.LState
var prompt string
var commands = map[string]bool{}
var aliases = map[string]string{}

func main() {
	parser := argparse.NewParser("hilbish", "A shell for lua and flower lovers")
	verflag := parser.Flag("v", "version", &argparse.Options{
		Required: false,
		Help: "prints hilbish version",
	})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(0)
	}

	if *verflag {
		fmt.Printf("Hilbish v%s\n", version)
		os.Exit(0)
	}

	os.Setenv("SHELL", os.Args[0])

	input, err := os.ReadFile(".hilbishrc.lua")
	if err != nil {
		input, err = os.ReadFile("/usr/share/hilbish/.hilbishrc.lua")
		if err != nil {
			fmt.Println("could not find .hilbishrc.lua or /usr/share/hilbish/.hilbishrc.lua")
			return
		}
	}

	homedir, _ := os.UserHomeDir()
	err = os.WriteFile(homedir + "/.hilbishrc.lua", input, 0644)
	if err != nil {
		fmt.Println("Error creating config file")
		fmt.Println(err)
		return
        }

	HandleSignals()
	LuaInit()

	for {
		//user, _ := user.Current()
		//dir, _ := os.Getwd()
		//host, _ := os.Hostname()

		//reader := bufio.NewReader(os.Stdin)

		//fmt.Printf(prompt)

		cmdString, err := readline.String(prompt)
		if err == io.EOF {
			fmt.Println("")
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		cmdString = strings.TrimSuffix(cmdString, "\n")
		err = l.DoString(cmdString)

		if err == nil { continue }

		cmdArgs := splitInput(cmdString)
		if len(cmdArgs) == 0 { continue }

		if aliases[cmdArgs[0]] != "" {
			cmdString = aliases[cmdArgs[0]] + strings.Trim(cmdString, cmdArgs[0])
			cmdArgs := splitInput(cmdString)
			execCommand(cmdArgs[0], cmdArgs[1:])
			continue
		}

		if commands[cmdArgs[0]] {
			err := l.CallByParam(lua.P{
				Fn: l.GetField(
					l.GetTable(
						l.GetGlobal("commanding"),
						lua.LString("__commands")),
					cmdArgs[0]),
				NRet: 0,
				Protect: true,
			}, luar.New(l, cmdArgs[1:]))
			if err != nil {
				// TODO: dont panic
				panic(err)
			}
			continue
		}
		switch cmdArgs[0] {
		case "exit":
			os.Exit(0)
		default:
			err := execCommand(cmdArgs[0], cmdArgs[1:])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}

func splitInput(input string) []string {
	quoted := false
	cmdArgs := []string{}
	sb := &strings.Builder{}

	for _, r := range input {
		if r == '"' {
			quoted = !quoted
			// dont add back quotes
			//sb.WriteRune(r)
		} else if !quoted && r == '~' {
			sb.WriteString(os.Getenv("HOME"))
		} else if !quoted && r == ' ' {
			cmdArgs = append(cmdArgs, sb.String())
			sb.Reset()
		} else {
			sb.WriteRune(r)
		}
	}
	if sb.Len() > 0 {
		cmdArgs = append(cmdArgs, sb.String())
	}

	readline.AddHistory(input)
	return cmdArgs
}
func execCommand(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout


	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
func HandleSignals() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
	}()
}

func LuaInit() {
	l = lua.NewState()

	l.OpenLibs()

	l.SetGlobal("prompt", l.NewFunction(hshprompt))
	l.SetGlobal("alias", l.NewFunction(hshalias))

	l.PreloadModule("fs", lfs.Loader)

	commander := cmds.New()
	commander.Events.On("commandRegister",
	func (cmdName string, cmd *lua.LFunction) {
		commands[cmdName] = true
		l.SetField(
			l.GetTable(l.GetGlobal("commanding"),
			lua.LString("__commands")),
			cmdName,
			cmd)
	})

	l.PreloadModule("commander", commander.Loader)

	l.DoString("package.path = package.path .. ';./libs/?/init.lua;/usr/share/hilbish/libs/?/init.lua'")

	err := l.DoFile("/usr/share/hilbish/preload.lua")
	if err != nil {
		err = l.DoFile("preload.lua")
		if err != nil {
			fmt.Fprintln(os.Stderr,
			"Missing preload file, builtins may be missing.")
		}
	}

	homedir, _ := os.UserHomeDir()
	err = l.DoFile(homedir + "/.hilbishrc.lua")
	if err != nil {
		panic(err)
	}
}
