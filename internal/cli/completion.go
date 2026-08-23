package cli

import "fmt"

const commandWords = "version completion login project repo pipeline build release doctor help"

func shellCompletion(shell string) (string, error) {
	switch shell {
	case "bash":
		return `# bash completion for cloudivision
_cloudivision() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  case "$prev" in
    project) COMPREPLY=( $(compgen -W "list create" -- "$cur") );;
    repo) COMPREPLY=( $(compgen -W "list add" -- "$cur") );;
    pipeline) COMPREPLY=( $(compgen -W "list" -- "$cur") );;
    build) COMPREPLY=( $(compgen -W "trigger list get logs watch cancel retry rerun" -- "$cur") );;
    release) COMPREPLY=( $(compgen -W "list get approve reject rollback" -- "$cur") );;
    completion) COMPREPLY=( $(compgen -W "bash zsh fish powershell" -- "$cur") );;
    *) COMPREPLY=( $(compgen -W "` + commandWords + ` --api-url --token --namespace --output --config" -- "$cur") );;
  esac
}
complete -F _cloudivision cloudivision
`, nil
	case "zsh":
		return `#compdef cloudivision
_cloudivision() {
  local -a commands
  commands=(` + commandWords + `)
  _arguments '*::command:->command'
  if [[ $state == command ]]; then _describe 'command' commands; fi
}
compdef _cloudivision cloudivision
`, nil
	case "fish":
		return `complete -c cloudivision -f
complete -c cloudivision -n '__fish_use_subcommand' -a '` + commandWords + `'
complete -c cloudivision -n '__fish_seen_subcommand_from build' -a 'trigger list get logs watch cancel retry rerun'
complete -c cloudivision -n '__fish_seen_subcommand_from release' -a 'list get approve reject rollback'
`, nil
	case "powershell":
		return `Register-ArgumentCompleter -Native -CommandName cloudivision -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  '` + commandWords + `'.Split(' ') | Where-Object { $_ -like "$wordToComplete*" } |
    ForEach-Object { [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_) }
}
`, nil
	default:
		return "", fmt.Errorf("unsupported shell %q; use bash, zsh, fish, or powershell", shell)
	}
}
