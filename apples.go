package main

import (
	"bufio"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"os"
	"os/exec"
	"strings"
)

// TODO turn this into a command line flag
const numWorkers int = 5

type applicationConfig struct {
	Common       common
	Applications map[string]application `toml:"applications"`
}

type common struct {
	SetupScripts []string `toml:"setup_scripts"`
}

type application struct {
	Steps   []string
	Command string
	Enabled bool
}

type outputLine struct {
	Prefix string
	Line   string
}

type commandSet struct {
	Steps      []string
	Prefix     string
	Output     chan outputLine
	ReturnCode chan bool
}

func (cs *commandSet) writeLine(line string) {
	cs.Output <- outputLine{
		Prefix: cs.Prefix,
		Line:   line,
	}
}

func (cs *commandSet) executeTask(cmd *exec.Cmd) error {
	outPipe, err := cmd.StdoutPipe()
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		cs.writeLine(fmt.Sprintf("Error reading command output %s", err))
		return err
	}
	cmd.Start()
	cs.readPipeOutput(outPipe)
	cs.readPipeOutput(errPipe)

	err = cmd.Wait()
	if err != nil {
		cs.writeLine(fmt.Sprintf("ERROR has occurred: %v", err))
		return err
	}
	cs.writeLine("Completed successfully")
	return nil
}

func (cs *commandSet) readPipeOutput(pipe io.ReadCloser) {
	stdout := bufio.NewReader(pipe)
	for {
		line, err := stdout.ReadString('\n')
		if err == nil || err == io.EOF {
			if len(line) > 0 {
				cs.writeLine(strings.TrimSuffix(line, "\n"))
			}
		}
		if err != nil {
			break
		}
	}
}

func (cs *commandSet) RunCommand() {
	for _, script := range cs.Steps {
		// To handle more complex commands maybec onsider switching to:
		// exec.Command("sh","-c", script)
		cs.writeLine(fmt.Sprintf("Running command `%s`", script))
		c := strings.Split(script, " ")
		cmd := exec.Command(c[0], c[1:]...)
		err := cs.executeTask(cmd)
		if err != nil {
			cs.ReturnCode <- false
			return
		}
		cs.ReturnCode <- true
	}
}

func RunCommandsInWorkerPool(commands []commandSet, numWorkers int) {
	cmdChan := make(chan commandSet)
	defer close(cmdChan)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for cmd := range cmdChan {
				cmd.RunCommand()
			}
		}()
	}

	for _, cs := range commands {
		cmdChan <- cs
	}
}

func writeLines(lines chan outputLine) {
	for line := range lines {
		fmt.Printf("%s: %s\n", line.Prefix, line.Line)
	}
}

func runEnabledTasks(apps map[string]application, lineChan chan outputLine, returnChan chan bool) bool {
	commands := make([]commandSet, 0, len(apps))
	for k, v := range apps {
		if v.Enabled {
			commands = append(commands, commandSet{
				Steps:      v.Steps,
				Prefix:     k,
				Output:     lineChan,
				ReturnCode: returnChan,
			})
		}
	}

	allPass := true
	numCommands := len(commands)
	go RunCommandsInWorkerPool(commands, numWorkers)
	for numCommands > 0 {
		numCommands--
		success := <-returnChan
		if !success {
			allPass = false
		}
	}
	return allPass
}

func writeProcFile(apps map[string]application) {
	f, err := os.Create("Procfile")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for k, v := range apps {
		if v.Enabled {
			f.WriteString(fmt.Sprintf("%s: %s\n", k, v.Command))
		}
	}
}

func main() {
	var config applicationConfig
	if _, err := toml.DecodeFile("applications.toml", &config); err != nil {
		fmt.Println(err)
		return
	}

	// Setup channel for capturing process output
	lineChan := make(chan outputLine)
	returnChan := make(chan bool)
	go writeLines(lineChan)
	defer close(lineChan)
	defer close(returnChan)

	// Run setup scripts
	setupCmd := commandSet{
		Steps:      config.Common.SetupScripts,
		Prefix:     "common setup",
		Output:     lineChan,
		ReturnCode: returnChan,
	}
	// TODO write synchronous version of this command
	go setupCmd.RunCommand()
	setupSuccessful := <-setupCmd.ReturnCode
	if !setupSuccessful {
		fmt.Printf("Bailing on application setup since setup scripts failed")
		os.Exit(1)
	}

	// Run application commands in parallel
	allPass := runEnabledTasks(config.Applications, lineChan, returnChan)
	if !allPass {
		fmt.Printf("Bailing on writing Procfile. Error in application scripts.")
		os.Exit(2)
	}

	// Write out Procfile to include enabled apps
	writeProcFile(config.Applications)
}
