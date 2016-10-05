# ARInGO
========

Simple Asterisk ARI connector from Go


## Installation ##

`go get github.com/cgrates/aringo`

## Support ##
Join [CGRateS](http://www.cgrates.org/ "CGRateS Website") on Google Groups [here](https://groups.google.com/forum/#!forum/cgrates "CGRateS on GoogleGroups").

## License ##
ARInGO is released under the [MIT License](http://www.opensource.org/licenses/mit-license.php "MIT License").
Copyright (C) ITsysCOM GmbH. All Rights Reserved.

## Sample usage code ##
```
package main

import (
        "fmt"
        "net/url"
        "time"

        "github.com/cgrates/aringo"
)

// listenAndServe will handle events received from ARInGO, mainly print them
func listenAndServe(evChan chan map[string]interface{}, errChan chan error) (err error) {
        for {
                select {
                case err = <-errChan:
                        return
                case astRawEv := <-evChan:
                        fmt.Printf("Received event from ARInGO: %+v\n", astRawEv) // your handler code goes here
                }
        }
        return fmt.Errorf("ListenAndServe out of select")
}

func main() {
        ariAddr := "127.0.0.1:8088"
        evChan := make(chan map[string]interface{}) // receive ARI events on this channel
        errChan := make(chan error)                 // receive ARI errors on this channel
        astConn, err := aringo.NewARInGO(fmt.Sprintf("ws://%s/ari/events?api_key=%s:%s&app=%s", ariAddr, "cgrates", "CGRateS.org", "cgrates_auth"),
                "http://cgrates.org", "cgrates", "CGRateS.org", "CGRateS 0.9.1~rc8", evChan, errChan, 5, 5) // connect to Asterisk ARI
        if err != nil {
                fmt.Printf("Error when connecting to ARI, <%s>", err.Error())
                return
        }
        go listenAndServe(evChan, errChan) // listen for events

        // Sample code for sending commands to ARI (originate a call)
        _, err = astConn.Call(aringo.HTTP_POST, fmt.Sprintf("http://%s/ari/channels?endpoint=PJSIP/1002&extension=1001", ariAddr),
                url.Values{})
        if err != nil {
                fmt.Printf("Error when sending originate to ARI, <%s>", err.Error())
                return
        }

        time.Sleep(20 * time.Second) // here just for testing, continue your code in the way you need
}
```