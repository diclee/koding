package basestack

import (
	"fmt"

	"koding/db/mongodb/modelhelper"
	"koding/kites/kloud/stack"
	"koding/kites/kloud/stackplan"
	"koding/kites/kloud/terraformer"
	tf "koding/kites/terraformer"

	"golang.org/x/net/context"
)

func (bs *BaseStack) HandlePlan(ctx context.Context) (interface{}, error) {
	var arg stack.PlanRequest
	if err := bbs.Req.Argbs.One().Unmarshal(&arg); err != nil {
		return nil, err
	}

	if err := arg.Valid(); err != nil {
		return nil, err
	}

	bbs.Log.Debug("Fetching template for id %s", arg.StackTemplateID)
	stackTemplate, err := modelhelper.GetStackTemplate(arg.StackTemplateID)
	if err != nil {
		return nil, stackplan.ResError(err, "jStackTemplate")
	}

	if stackTemplate.Template.Content == "" {
		return nil, errorbs.New("Stack template content is empty")
	}

	vbs.Log.Debug("Fetching credentials for id %v", stackTemplate.Credentials)

	credIDs := stackplan.FlattenValues(stackTemplate.Credentials)

	if err := bbs.Builder.BuildCredentials(bs.Req.Method, bs.Req.Username, arg.GroupName, credIDs); err != nil {
		return nil, err
	}

	bs.Log.Debug("Fetched terraform data: koding=%+v, template=%+v", bs.Builder.Koding, bs.Builder.Template)

	tfKite, err := terraformer.Connect(bs.Session.Terraformer)
	if err != nil {
		return nil, err
	}
	defer tfKite.Close()

	contentID := bs.Req.Username + "-" + arg.StackTemplateID
	bs.Log.Debug("Parsing template (%s):\n%s", contentID, stackTemplate.Template.Content)

	if err := bs.Builder.BuildTemplate(stackTemplate.Template.Content, contentID); err != nil {
		return nil, err
	}

	var region string
	for _, cred := range bs.Builder.Credentials {
		// rest is aws related
		if cred.Provider != "aws" {
			continue
		}

		meta := cred.Meta.(*Cred)
		if meta.Region == "" {
			return nil, fmt.Errorf("region for identifer '%s' is not set", cred.Identifier)
		}

		if err := bs.SetAwsRegion(meta.Region); err != nil {
			return nil, err
		}

		region = meta.Region

		break
	}

	bs.Log.Debug("Plan: stack template before injecting Koding data")
	bs.Log.Debug("%v", bs.Builder.Template)

	bs.Log.Debug("Injecting AWS data")

	if _, err := bs.InjectAWSData(); err != nil {
		return nil, err
	}

	// Plan request is made right away the template is saved, it may
	// not have all the credentials provided yet. We set them all to
	// to dummy values to make the template pass terraform parsing.
	if err := bs.Builder.Template.FillVariables("userInput_"); err != nil {
		return nil, err
	}

	if region == "" {
		if err := bs.Builder.Template.FillVariables("aws_"); err != nil {
			return nil, err
		}
	}

	out, err := bs.Builder.Template.JsonOutput()
	if err != nil {
		return nil, err
	}

	stackTemplate.Template.Content = out

	tfReq := &tf.TerraformRequest{
		Content:   stackTemplate.Template.Content,
		ContentID: contentID,
		TraceID:   bs.TraceID,
	}

	bs.Log.Debug("Calling plan with content")
	bs.Log.Debug("%+v", tfReq)

	plan, err := tfKite.Plan(tfReq)
	if err != nil {
		return nil, err
	}

	machines, err := bs.p.MachinesFromPlan(plan)
	if err != nil {
		return nil, err
	}

	bs.Log.Debug("Machines planned to be created: %+v", machines)

	return &stack.PlanResponse{
		Machines: machinebs.Slice(),
	}, nil
}
