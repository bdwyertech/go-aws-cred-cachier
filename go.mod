module github.com/bdwyertech/go-aws-cred-cachier/aws-cred-cachier

go 1.14

require (
	bou.ke/monkey v1.0.2
	github.com/aws/aws-sdk-go v1.30.19
	github.com/gofrs/flock v0.7.1
	github.com/jcelliott/lumber v0.0.0-20160324203708-dd349441af25 // indirect
	github.com/kami-zh/go-capturer v0.0.0-20171211120116-e492ea43421d
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/sdomino/scribble v0.0.0-20191024200645-4116320640ba
	github.com/stretchr/testify v1.5.1
)

replace github.com/gofrs/flock => github.com/azr/flock v0.7.2-0.20200319085905-0eda2671edf3
