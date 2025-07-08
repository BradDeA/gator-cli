# Gator CLI blog aggregator program

You will need postgres SQL server and go installed on your machine.

### Install Postgres
`sudo apt update`
`sudo apt install postgresql postgresql-contrib`

### Install Go
You can find Go's offical installation process [here](https://go.dev/doc/install)

or

Use the webi installation [here](https://webinstall.dev/golang/)

### Download and install gator CLI
Download or pull the gator cli package from github to build yourself

or

Use `go install [github link]` to install using go

### Config file setup
Create a file in your home directory named `.gatroconfig.json`

Ensure postgres is running with
`sudo service postgresql start`

Once postgres is running, you must determie your connection string, it should be formatted as such:
`protocol://username:password@host:port/database`

An exmaple of what your connection string might look like:
`postgres://username:password@localhost:5432/gator`

`5432` is the default port for postgres and gator is the database used by the program. Since this only runs locally your host will probably be localhost.

Check your connection string is correct and working by using `psql "postgres://username:password@localhost:5432/gator"` using your own connection string of course.

Finally, once your connection string is working and correct, add this to the config file that was created in the first step, its structure is as follows

`{
  "db_url": "connection_string_goes_here",
  "current_user_name": "username_goes_here"
}`

You dont need to set a current_user_name, the program will handle logging in and seting the current user when a new user is registered, or when you login as an existing user, the following is valid for the program:

`{
  "db_url": "connection_string_goes_here"
}`


### Usage

Once you have installed the necessary programs, started postgres and created your config file, you should be able to use gator!

The following are some important commands you should know: <br>
`gator register "username"` -> register a new user and login as that user <br>
`gator login "username"` -> login as specified user <br>
`gator addfeed "feed name" "feed URL"` -> add an RSS feed to gator <br>
`gator agg "time"` -> aggregates feeds on a set interval, meant to be run in the background to pull posts from the RSS feeds, "time" must formatted like the following: `5s, 5m, 5h` <br>
`gator follow "URL"` -> follows the specified feed using it's URL <br>
`gator browse [optional limit]` -> will display the posts from currently aggregated feeds, you may optionally set how many posts you want returned. <br>
