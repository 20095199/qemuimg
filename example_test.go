package qemuimg

import (
	"context"
	"fmt"
)

func ExampleQemuImg_CreateCommand() {
	runner := &fakeRunner{handler: fixtureHandler(modernHelp)}
	q := New(WithRunner(runner))

	command, err := q.CreateCommand(context.Background(), CreateRequest{
		Filename: "/data/base.qcow2",
		Format:   FormatQcow2,
		Size:     "10G",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(command.String())
	// Output: qemu-img create -f qcow2 /data/base.qcow2 10G
}

func ExampleOptions() {
	options := NewOptions(
		Option{Key: "compat", Value: "1.1"},
		Option{Key: "lazy_refcounts", Value: "on"},
	)

	fmt.Println(options.String())
	// Output: compat=1.1,lazy_refcounts=on
}
