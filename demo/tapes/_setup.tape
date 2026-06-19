# Shared VHS settings + hidden sandbox prep. Every per-command tape does
# `Source demo/tapes/_setup.tape`. REAL recordings of the actual TUI: the
# sandbox sources the binary against a throwaway HOME and points it at a local
# Groq mock (no network, no API key, no quota — see setup-sandbox.sh).

Set Shell bash
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 15
Set Width 1280
Set Padding 16
# No window chrome: a real terminal has no margin band or rounded frame. The
# default VHS margin fills with its own dark color, which clashes with the TUI's
# background — so drop the margin and let the whole surface be the TUI's own
# background, exactly like a real terminal. Background = harmonized theme BG
# (#0e1016); keep this in sync with the theme in setup-sandbox.sh.
Set Margin 0
Set BorderRadius 0
Set MarginFill "#0e1016"
Set Theme { "name": "commitcraft", "background": "#0e1016", "foreground": "#cfd3d8", "black": "#0e1016", "white": "#cfd3d8", "cursor": "#c5a3ff", "selection": "#161922" }
Set TypingSpeed 45ms
Set PlaybackSpeed 1.0

Hide
# Force 24-bit color so the TUI's hex (truecolor) theme renders EXACTLY as in a
# real terminal, instead of being downsampled to the recorder's 256/16 palette.
Type "export COLORTERM=truecolor TERM=xterm-256color" Enter
Type "source demo/setup-sandbox.sh" Enter
Type `export PS1='\[\e[38;5;141m\]api-service\[\e[0m\] \[\e[38;5;78m\]❯\[\e[0m\] '` Enter
Type "clear" Enter
Show
Sleep 600ms
