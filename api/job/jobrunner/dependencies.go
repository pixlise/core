package jobrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func installPythonLibs(pythonPath string) error {
	f, err := os.Stat("requirements.txt")
	if err == nil && !f.IsDir() && f.Name() == "requirements.txt" {
		// Run pip
		pipPath := filepath.Join(pythonPath, "pip")
		fmt.Printf("Running %v to install requirements.txt...\n", pipPath)
		out, err := runCommand(pipPath, []string{"install", "-r", "requirements.txt"})
		if err != nil {
			fmt.Printf("Failed to install python libraries:\n%v\n", string(out))
			return err
		}
		fmt.Println("  ...Success")
	} else {
		fmt.Println("requirements.txt not found")
	}

	// No requirements.txt found or it worked... no errors!
	return nil
}

func installLuaLibs() error {
	// If we're dealing with a rockspec file, treat it as such
	allargs := [][]string{}

	f, err := os.Stat("requirements.rockspec")
	if err == nil && !f.IsDir() && f.Name() == "requirements.rockspec" {
		allargs = append(allargs, []string{"luarocks-5.3", "install", "requirements.rockspec"})
	} else {
		// See if there's a lua-requirements.txt, we'll read it line-by-line in that case and install each
		b, err := os.ReadFile("lua-requirements.txt")
		if err == nil {
			lines := strings.Split(string(b), "\n")
			for _, line := range lines {
				allargs = append(allargs, []string{"luarocks-5.3", "install", line})
			}
		}
	}

	// Run all commands, return if an error happens
	for _, args := range allargs {
		fmt.Printf("Executing: %v\n", strings.Join(args, " "))
		out, err := runCommand(args[0], args[1:])
		if err != nil {
			fmt.Printf("Error while installing lua library [%v]: %v\n", strings.Join(args, ","), string(out))
			return err
		}
	}

	return nil
}
