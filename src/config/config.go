// from git.tcp.direct/kayos/CokePlate
package config

import (
	"bytes"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strings"
)

//////////// Application version information //
const (
	Version = "0.1"
	Title   = "HellPot"
)

var appLabel string = Title + " " + Version

// --------------------------------------------

/*
 define _Exported_ variables below (add additional configuration directives here)

 ** Note for those new to golang: **

   capitalizing the first letter allows you to call it from different packages
   we later reference these with something like this:

       import "vx-search/src/config"

       if config.Debug {
         //debug shit here
       }

       net.http.ListenAndServe(fmt.Sprintf("%s:%d", config.BindAddr, config.BindPort)

   -----------------------------------------------------------------------------------

 ** Additional notes: **

   Once you declare the variables here you must also load the values into them
   search for "viper.GetString" to see how we do that in this file

   See: func associate()

*/
var (
	Debug     bool = false
	LogDir    string
	Banner    string
	DataDir   string
	Databases []string
	//Color bool
	BindAddr string
	BindPort string
	Paths    []string
)

// -----------------------------------------------------------------

var (
	f   *os.File
	err error

	// from cli flags
	forcedebug   bool = false
	customconfig bool

	/*Config (for viper)
		this is an exported variable that merely points to the underlying viper config instance
	        This will allow us to reference the viper type from any other package if we need to */
	Config *viper.Viper

	log zerolog.Logger

	configLocations []string
	home            string

	// for buffering our config messages so they spit out _after_ the banner
	logBuffer *lineBuffer

	// configLog *bufio.Writer
)

type lineBuffer struct {
	lines []string // our buffered log lines
}

func (log *lineBuffer) Write(data []byte) (int, error) {
	line := string(data)
	log.lines = append(log.lines, line)
	return len(data), nil
}

func acquireClue() {
	// define proper console output before we determine a log file location
	log = preLog()

	if home, err = os.UserHomeDir(); err != nil {
		log.Fatal().Err(err).Msg("failed to determine user's home directory, we will not be able to load our configuration if it is stored there!")
	}

	///////////////////////////////////////////////////////////////////
	// define locations we will look for our toml configuration file //
	///////////////////////////////////////////////////////////////////

	// e.g: /home/fuckhole/.jonesapp/config.toml
	configLocations = append(configLocations, home+"/."+Title+"/")

	// e.g: /etc/jonesapp/config.toml
	configLocations = append(configLocations, "/etc/"+Title+"/")

	// e.g: /home/fuckhole/Workshop/jonesapp/config.toml
	configLocations = append(configLocations, "./")

	// e.g: /home/fuckhole/Workshop/config.toml
	configLocations = append(configLocations, "../")

	// e.g: /home/fuckhole/Workshop/jonesapp/.config/config.toml
	configLocations = append(configLocations, "./.config/")

	// ----------------------------------------------------------------

}

/* preLog is to temporarily define pretty printing before we finish initializing our json logger
   note that here we are piping it to our configLog (see: PrintConfigLog)
*/
func preLog() zerolog.Logger {
	if forcedebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// initiate an instance of our custom writer
	logBuffer = new(lineBuffer)
	return zerolog.New(zerolog.ConsoleWriter{Out: logBuffer}).With().Timestamp().Logger()
}

/*PrintConfigLog (for debug)
  here we implement a buffer for the log lines generated by our configuration engine
  We use this to assure that our log output appears _after_ the banner output instead of before it */
func PrintConfigLog() {
	for _, line := range logBuffer.lines {
		print(line)
	}
}

// Blueprint will initialize our toml configuration engine and define our default configuration values which can be written to a new configuration file if desired
func Blueprint() {

	// handle command line override of the config file location
	// e.g: ./app -c /home/fuckhole/jones.toml
	for i, arg := range os.Args {
		switch arg {
		case "-c":
			if len(os.Args) <= i-1 {
				panic("syntax error! expected file after -c")
			}

			if f, err = os.Open(os.Args[i+1]); err != nil {
				println("Error opening specified config file: " + os.Args[i+1])
				panic("config file open fatal error: " + err.Error())
			}

			buf, err := ioutil.ReadAll(f)
			err2 := Config.ReadConfig(bytes.NewBuffer(buf))

			if err != nil || err2 != nil {
				println("Error reading specified config file: " + os.Args[i+1])
				if err != nil {
					panic("config file read fatal error: " + err.Error())
				} else {
					panic("config file read fatal error: " + err2.Error())
				}
			}

			customconfig = true

		case "-d":
			forcedebug = true
			Debug = true
		}
	}

	if customconfig {
		associate()
		return
	}

	acquireClue()

	Config = viper.New()

	///////////////////// defaults //////
	defName := appLabel

	defLogger := map[string]interface{}{
		"debug":         true,
		"log_directory": "./.logs/",
	}

	defHTTP := map[string]interface{}{
		"bind_addr": "127.0.0.1",
		"bind_port": "8080",
		"paths": []string{
			"wp-login.php",
			"wp-login",
		},
	}

	/*
		defData := map[string]interface{}{
			"directory": "./.data/",
		}

		// here we are defining a generic category as an example
		defCategory := map[string]interface{}{
			"shouldistay": true,
			"shouldigo":   false,
			"optics":      "ironsights",
			"fucksgiven":  0,
			"admins": []string{"Satan", "Yahweh", "FuckholeJones"},
		}
	*/

	Config.SetDefault("name", defName)
	Config.SetDefault("logger", defLogger)
	Config.SetDefault("http", defHTTP)
	//Config.SetDefault("database", defData)
	//Config.SetDefault("category", defCategory)

	Config.SetConfigType("toml")
	Config.SetConfigName("config")

	// iter through our previously defined configuration file locations and add them to the paths searched by viper
	for _, loc := range configLocations {
		Config.AddConfigPath(loc)
		log.Debug().Str("directory", loc).Msg("New config file location registered")
	}

	// locate and read the config file
	err = Config.ReadInConfig()

	// if we can't locate a config file, write one in the same directory as we're executing in based on the maps we defined above
	if err != nil {
		if strings.Contains(err.Error(), "Not Found in") {
			log.Warn().
				Msg("Config file not found! Writing new config to config.toml")
		}

		if err := Config.SafeWriteConfigAs("./config.toml"); err != nil {
			log.Fatal().Err(err).Msg("Error writing new configuration file")
		}
	} else {
		log.Info().Str("file", Config.ConfigFileUsed()).Msg("Successfully loaded configuration file")
	}

	associate()
}

// Assignments is where we assign the values of our variables with our configuration values from defaults and/or the config file
func associate() {

	// define loglevel of application
	if !Config.GetBool("logger.debug") && !forcedebug {
		Debug = false
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		Debug = true
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	for _, key := range Config.AllKeys() {
		log.Debug().Str("key", key).Msg("LOAD_CONFIG_DIRECTIVE")
	}

	// location for our log files generated at runtime
	LogDir = Config.GetString("logger.log_directory")

	// bitcask database parameters (casket)
	//DataDir = Config.GetString("database.directory")
	//Databases = Config.GetStringSlice("database.databases")

	// HellPot specific directives
	BindAddr = Config.GetString("http.bind_addr")
	BindPort = Config.GetString("http.bind_port")
	Paths = Config.GetStringSlice("http.paths")
}
