package basestack

import (
	"bytes"
	"fmt"
	"text/template"

	"koding/kites/kloud/stack"
	"koding/kites/kloud/terraformer"
	tf "koding/kites/terraformer"

	"golang.org/x/net/context"
)

// Bootstrap
func (bs *BaseStack) HandleBootstrap(context.Context) (interface{}, error) {
	var arg stack.BootstrapRequest
	if err := bs.Req.Args.One().Unmarshal(&arg); err != nil {
		return nil, err
	}

	return bs.bootstrap(&arg)
}

func (bs *BaseStack) bootstrap(arg *stack.BootstrapRequest) (interface{}, error) {
	if err := arg.Valid(); err != nil {
		return nil, err
	}

	if arg.Destroy {
		bs.Log.Debug("Bootstrap destroy is called")
	} else {
		bs.Log.Debug("Bootstrap apply is called")
	}

	if err := bs.Builder.BuildCredentials(bs.Req.Method, bs.Req.Username, arg.GroupName, arg.Identifiers); err != nil {
		return nil, err
	}

	bs.Log.Debug("Connecting to terraformer kite")

	tfKite, err := terraformer.Connect(bs.Session.Terraformer)
	if err != nil {
		return nil, err
	}
	defer tfKite.Close()

	bs.Log.Debug("Iterating over credentials")

	for _, cred := range bs.Builder.Credentials {
		if cred.Provider != bs.Planner.Provider {
			continue
		}

		templates, err := bs.Stack.BootstrapTemplates(cred.Meta)
		if err != nil {
			return nil, err
		}

		// TODO(rjeczalik): refactor to split Destroy code flow from Apply one
		// - for readibility.

		uniq := make(map[string]interface{})
		for _, tmpl := range templates {
			if tmpl.Key == "" {
				tmpl.Key = bs.Planner.Provider + "-" + arg.GroupName + "-" + cred.Identifier
			}
			uniq[tmpl.Key] = struct{}{}

			if arg.Destroy {
				bs.Log.Info("Destroying bootstrap resources belonging to identifier '%s'", cred.Identifier)

				_, err := tfKite.Destroy(&tf.TerraformRequest{
					ContentID: contentID,
					TraceID:   bs.TraceID,
				})
				if err != nil {
					return nil, err
				}

				cred.Meta = bs.Provider.NewCredential

				// TODO(rjeczalik): find out how to update stuff

				meta.ResetBootstrap()
			} else {
				bs.Log.Info("Creating bootstrap resources belonging to identifier '%s'", cred.Identifier)
				bs.Log.Debug("Bootstrap template:")
				bs.Log.Debug("%s", bootstrapTemplate)

				if err := bs.Builder.BuildTemplate(bootstrapTemplate, contentID); err != nil {
					return nil, err
				}

				finalBootstrap, err := bs.Builder.Template.JsonOutput()
				if err != nil {
					return nil, err
				}

				bs.Log.Debug("Final bootstrap template:")
				bs.Log.Debug("%s", finalBootstrap)

				state, err := tfKite.Apply(&tf.TerraformRequest{
					Content:   finalBootstrap,
					ContentID: contentID,
					TraceID:   bs.TraceID,
				})
				if err != nil {
					return nil, err
				}

				bs.Log.Debug("[%s] state.RootModule().Outputs = %+v\n", cred.Identifier, state.RootModule().Outputs)

				if err := bs.Builder.Object.Decode(state.RootModule().Outputs, meta); err != nil {
					return nil, err
				}

				bs.Log.Debug("[%s] resp = %+v\n", cred.Identifier, meta)

				if err := meta.BootstrapValid(); err != nil {
					return nil, fmt.Errorf("invalid bootstrap metadata for %q: %s", cred.Identifier, err)
				}
			}
		}

		if err != nil {
			return nil, err
		}

		bs.Log.Debug("[%s] Bootstrap response: %+v", cred.Identifier, meta)

		datas := map[string]interface{}{
			cred.Identifier: meta,
		}

		if err := bs.Builder.CredStore.Put(bs.Req.Username, datas); err != nil {
			return nil, err
		}
	}

	return true, nil
}

func newTemplate(awsData *awsTemplateData) (string, error) {
	var buf bytes.Buffer

	if err := awsBootstrap.Execute(&buf, awsData); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// awsTemplateData is being used to format the bootstrap before we pass it to
// terraformer
type awsTemplateData struct {
	AvailabilityZone string
	KeyPairName      string
	PublicKey        string
	EnvironmentName  string
}

var awsBootstrap = template.Must(template.New("").Parse(`{
    "provider": {
        "aws": {
            "access_key": "${var.aws_access_key}",
            "secret_key": "${var.aws_secret_key}",
            "region": "${var.aws_region}"
        }
    },
    "output": {
        "vpc": {
            "value": "${aws_vpc.vpc.id}"
        },
        "cidr_block": {
            "value": "${aws_vpc.vpc.cidr_block}"
        },
        "rtb": {
            "value": "${aws_vpc.vpc.main_route_table_id}"
        },
        "acl": {
            "value": "${aws_vpc.vpc.default_network_acl_id}"
        },
        "igw": {
            "value": "${aws_internet_gateway.main_vpc_igw.id}"
        },
        "subnet": {
            "value": "${aws_subnet.main_koding_subnet.id}"
        },
        "sg": {
            "value": "${aws_security_group.allow_all.id}"
        },
        "ami": {
            "value": "${lookup(var.aws_amis, var.aws_region)}"
        },
        "key_pair": {
            "value": "${aws_key_pair.koding_key_pair.key_name}"
        }
    },
    "resource": {
        "aws_vpc": {
            "vpc": {
                "cidr_block": "${var.cidr_block}",
                "tags": {
                    "Name": "${var.environment_name}"
                }
            }
        },
        "aws_internet_gateway": {
            "main_vpc_igw": {
                "tags": {
                    "Name": "${var.environment_name}"
                },
                "vpc_id": "${aws_vpc.vpc.id}"
            }
        },
        "aws_subnet": {
            "main_koding_subnet": {
                "availability_zone": "{{.AvailabilityZone}}",
                "cidr_block": "${var.cidr_block}",
                "map_public_ip_on_launch": true,
                "tags": {
                    "Name": "${var.environment_name}",
                    "subnet": "public"
                },
                "vpc_id": "${aws_vpc.vpc.id}"
            }
        },
        "aws_route_table": {
            "public": {
                "route": {
                    "cidr_block": "0.0.0.0/0",
                    "gateway_id": "${aws_internet_gateway.main_vpc_igw.id}"
                },
                "tags": {
                    "Name": "${var.environment_name}",
                    "routeTable": "koding",
                    "subnet": "public"
                },
                "vpc_id": "${aws_vpc.vpc.id}"
            }
        },
        "aws_route_table_association": {
            "public-1": {
                "route_table_id": "${aws_route_table.public.id}",
                "subnet_id": "${aws_subnet.main_koding_subnet.id}"
            }
        },
        "aws_security_group": {
            "allow_all": {
                "description": "Allow all inbound and outbound traffic",
                "ingress": {
                    "from_port": 0,
                    "to_port": 0,
                    "protocol": "-1",
                    "cidr_blocks": ["0.0.0.0/0"],
                    "self": true
                },
                "egress": {
                    "from_port": 0,
                    "to_port": 0,
                    "protocol": "-1",
                    "self": true,
                    "cidr_blocks": ["0.0.0.0/0"]
                },
                "name": "allow_all",
                "tags": {
                    "Name": "${var.environment_name}"
                },
                "vpc_id": "${aws_vpc.vpc.id}"
            }
        },
        "aws_key_pair": {
            "koding_key_pair": {
                "key_name": "${var.key_name}",
                "public_key": "${var.public_key}"
            }
        }
    },
    "variable": {
        "cidr_block": {
            "default": "10.0.0.0/16"
        },
        "environment_name": {
            "default": "Koding-Bootstrap"
        },
        "aws_availability_zones": {
            "default": {
                "ap-northeast-1": "ap-northeast-1b",
                "ap-southeast-1": "ap-southeast-1b",
                "ap-southeast-2": "ap-southeast-2b",
                "eu-central-1": "eu-central-1b",
                "eu-west-1": "eu-west-1b",
                "sa-east-1": "sa-east-1b",
                "us-east-1": "us-east-1b",
                "us-west-1": "us-west-1b",
                "us-west-2": "us-west-2b"
            }
        },
        "aws_amis": {
            "default": {
                "ap-northeast-1": "ami-9e5cff9e",
                "ap-southeast-1": "ami-ec7879be",
                "ap-southeast-2": "ami-2fce8b15",
                "eu-central-1": "ami-60f9c27d",
                "eu-west-1": "ami-7c4b0a0b",
                "sa-east-1": "ami-cd9518d0",
                "us-east-1": "ami-cf35f3a4",
                "us-west-1": "ami-b33dccf7",
                "us-west-2": "ami-8d5b5dbd"
            }
        },
		"key_name": {
			"default": "{{.KeyPairName}}"
		},
		"public_key": {
			"default": "{{.PublicKey}}"
		},
		"environment_name": {
			"default": "{{.EnvironmentName}}"
		}
    }
}`))
