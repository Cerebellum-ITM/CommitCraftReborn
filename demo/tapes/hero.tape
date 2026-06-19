Output demo/gifs/hero.gif
Set Height 760
Source demo/tapes/_setup.tape

# Launch the TUI (History tab, then start a new commit).
Type "commitcraft" Enter
Sleep 1.8s
Type "n"
Sleep 1.2s

# Pick the commit type (Ctrl+T → filter → enter).
Ctrl+T
Sleep 1s
Type "ADD"
Sleep 700ms
Enter
Sleep 1s

# Pick the scope (Ctrl+P → into internal/ → pick the api package).
Ctrl+P
Sleep 1.1s
Tab
Sleep 900ms
Enter
Sleep 1.1s

# Describe the change, then let CommitCraft generate the message (^W).
Type "retry transient Groq 429s with exponential backoff"
Sleep 900ms
Ctrl+W
Sleep 5s

