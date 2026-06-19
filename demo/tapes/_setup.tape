# Shared VHS settings + hidden sandbox prep. Every per-command tape does
# `Source demo/tapes/_setup.tape` so these live in one place. REAL mode: the
# sandbox sources the actual binary against a throwaway HOME (no Groq, no secrets).

Set Shell bash
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1120
Set Padding 22
Set Margin 16
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set PlaybackSpeed 1.0

Hide
Type "source demo/setup-sandbox.sh" Enter
Type `export PS1='\[\e[38;5;141m\]api-service\[\e[0m\] \[\e[38;5;78m\]❯\[\e[0m\] '` Enter
Type "clear" Enter
Show
Sleep 700ms
