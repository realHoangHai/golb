package config

type Replica struct {
	Url      string            `yaml:"url"`
	Metadata map[string]string `yaml:"metadata"`
}

type Service struct {
	Name string `yaml:"name"`

	// A prefix matcher to select service based on the path part of the url
	// Note(self): The matcher could be more sophisticated (i.e Regex based,
	// subdomain based), but for the purposes of simplicity let's think about this
	// later, and it could be a nice contribution to the project.
	Matcher string `yaml:"matcher"`

	// Strategy is the load balancing strategy used for this service.
	Strategy string `yaml:"strategy"`

	Replicas []Replica `yaml:"replicas"`
}

// Config is a representation of the configuration given from a config source.
type Config struct {
	Port     int       `yaml:"port"`
	Services []Service `yaml:"services"`

	// Name of strategy to be used in load balancing between instances
	Strategy string `yaml:"strategy"`
}
