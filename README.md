# Interactive diff

A simple tool to compare Frontend and DCR interactives as we migrate them.

To run, simply grab the latest release and execute it:

    $ PORT=8080 ./interactive-diff

## Development

Use Gin for watch/reload:

    $ go get github.com/codegangsta/gin
    $ gin -all run main.go
