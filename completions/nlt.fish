# Completions for the nlt alias (note list -t).
# Each argument is a tag value; fish filters candidates by the current token prefix.
complete -c nlt -f -a '(note tags 2>/dev/null | string replace -r "^#" "")' -d Tag
