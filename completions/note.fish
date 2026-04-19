# Fish completions for note.
# Install: ln -s (realpath completions/note.fish) ~/.config/fish/completions/note.fish

function __note_completing_tag
    set -l tok (commandline -ct)
    string match -q '#*' -- $tok; or string match -q ',,*' -- $tok
end

# Disable file completions globally for note.
complete -c note -f

# Top-level actions.
complete -c note -n 'not __fish_seen_subcommand_from add list tags edit delete' \
    -a 'add\tAppend\ a\ note list\tList\ notes tags\tList\ tags edit\tOpen\ editor delete\tDelete\ notes'

# Tag completion for `note add` — fires when current token starts with # or ,,
complete -c note -n '__fish_seen_subcommand_from add; and __note_completing_tag' \
    -a '(note completions (commandline -ct) 2>/dev/null)'

# Tag completion for `note list -t`
complete -c note -n '__fish_seen_subcommand_from list' \
    -s t -l tag -r -a '(note tags 2>/dev/null | string replace "#" "")'
