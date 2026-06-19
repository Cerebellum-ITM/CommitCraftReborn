# Shared VHS settings + hidden sandbox prep. Every per-command tape does
# `Source demo/tapes/_setup.tape`. REAL recordings of the actual TUI: the
# sandbox sources the binary against a throwaway HOME and points it at a local
# Groq mock (no network, no API key, no quota — see setup-sandbox.sh).

Set Shell bash
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 15
Set Width 1280
Set Padding 20
Set Margin 14
Set BorderRadius 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set PlaybackSpeed 1.0

Hide
Type "source demo/setup-sandbox.sh" Enter
Type `export PS1='\[\e[38;5;141m\]api-service\[\e[0m\] \[\e[38;5;78m\]❯\[\e[0m\] '` Enter
Type "clear" Enter
Show
Sleep 600ms
