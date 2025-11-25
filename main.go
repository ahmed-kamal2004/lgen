package main

import (
	"generator/load/cmd"
)

func main(){
	rootCmd := cmd.NewRootCommand()
 	if err := rootCmd.Execute(); err != nil {
        panic(err)
    }
}