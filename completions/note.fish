# Disable file completions for the top-level command
complete -c note -f

# Resolve the notes file from the current commandline flags
function __note_file
    set -l tokens (commandline -opc)
    set -l file ""
    for i in (seq 1 (count $tokens))
        switch $tokens[$i]
            case '-f' '--file'
                set -l next (math $i + 1)
                if test $next -le (count $tokens)
                    set file $tokens[$next]
                end
        end
    end
    if test -z "$file"
        if set -q NOTE_FILE
            echo $NOTE_FILE
        else
            echo "$HOME/notes/quick-notes.md"
        end
    else
        echo $file
    end
end

function __note_tags
    grep -oE '#[a-zA-Z0-9_]+' (__note_file) 2>/dev/null | string replace '#' '' | sort -u
end

function __note_ids
    grep -oE '\[[0-9]+\]' (__note_file) 2>/dev/null | string replace -ra '[\[\]]' '' | sort -un
end

# Global flags
complete -c note -s f -l file -r -d 'Target a specific notes file' -F
complete -c note -s t -l tag  -r -d 'Filter by tag(s), comma-separated' -a "(__note_tags)"
complete -c note -s h -l help -d 'Show help'

# Actions (only when no action has been given yet)
set -l actions add list tags edit delete
complete -c note -n "not __fish_seen_subcommand_from $actions" -a add    -d 'Append a timestamped note'
complete -c note -n "not __fish_seen_subcommand_from $actions" -a list   -d 'Display all notes'
complete -c note -n "not __fish_seen_subcommand_from $actions" -a tags   -d 'List all tags'
complete -c note -n "not __fish_seen_subcommand_from $actions" -a edit   -d 'Open notes in nvim (or edit by id)'
complete -c note -n "not __fish_seen_subcommand_from $actions" -a delete -d 'Delete notes by id'

# add: complete #tag and ,,tag[,tag].. tokens.
# For ,,tag: a custom Tab binding handles single-match insertion without the
# trailing space fish would normally add, so the user can chain: ,,music,mo<tab>.
# For multiple matches fish uses the -a candidates and inserts the common prefix
# (which also has no trailing space), so the binding falls through for that case.
complete -c note -n "__fish_seen_subcommand_from add" \
    -a "(set -l tok (commandline -ct)
        if string match -q '#*' -- \$tok; or string match -q ',,*' -- \$tok
            note completions --file (__note_file) \$tok 2>/dev/null
        end)"


# list: -t / --tag with tag completion
complete -c note -n "__fish_seen_subcommand_from list" -s t -l tag -r -d 'Filter by tag(s)' -a "(__note_tags)"

# edit / delete: complete note IDs
complete -c note -n "__fish_seen_subcommand_from edit delete" -a "(__note_ids)" -d 'Note id'
