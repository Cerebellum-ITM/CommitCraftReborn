Output demo/gifs/tags.gif
Set Height 340
Source demo/tapes/_setup.tape

Type "# The commit-type tags 'ai generate' accepts (default + global + local):" Enter
Sleep 1.2s
Type "commitcraft ai list-tags | jq -c '.[] | {tag, source}'" Sleep 400ms Enter
Sleep 3.2s
