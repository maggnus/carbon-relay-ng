package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Conn struct {
	conn        *net.TCPConn
	buffered    *bufio.Writer
	shutdown    chan bool
	In          chan []byte
	dest        *Destination // which dest do we correspond to
	up          bool
	checkUp     chan bool
	updateUp    chan bool
	flush       chan bool
	flushErr    chan error
	periodFlush time.Duration
}

func NewConn(addr string, dest *Destination, periodFlush time.Duration) (*Conn, error) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	laddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0")
	conn, err := net.DialTCP("tcp", laddr, raddr)
	if err != nil {
		return nil, err
	}
	connObj := &Conn{
		conn:        conn,
		buffered:    bufio.NewWriter(conn),
		shutdown:    make(chan bool),
		In:          make(chan []byte),
		dest:        dest,
		up:          true,
		checkUp:     make(chan bool),
		updateUp:    make(chan bool),
		flush:       make(chan bool),
		flushErr:    make(chan error),
		periodFlush: periodFlush,
	}

	go connObj.HandleData()
	go connObj.HandleStatus()
	return connObj, nil
}

func (c *Conn) isAlive() bool {
	return <-c.checkUp
}

func (c *Conn) HandleStatus() {
	for {
		select {
		case c.up = <-c.updateUp:
			//log.Println("conn.up is now", c.up)
		case c.checkUp <- c.up:
			//log.Println("query for conn.up, responded with", c.up)
		}
	}
}

func (c *Conn) HandleData() {
	periodFlush := c.periodFlush
	tickerFlush := time.NewTicker(periodFlush)

	for {
		select {
		case buf := <-c.In:
			log.Printf("%s conn writing %s\n", c.dest.Addr, string(buf))
			buf = append(buf, '\n')
			n, err := c.Write(buf)
			errBecauseTruncated := false
			if err == nil && len(buf) != n {
				errBecauseTruncated = true
				c.dest.numErrTruncated.Add(1)
				err = errors.New(fmt.Sprintf("truncated write: %s", string(buf)))
			}
			if err != nil {
				if !errBecauseTruncated {
					c.dest.numErrWrite.Add(1)
				}
				log.Println(c.dest.Addr + " " + err.Error())
				fmt.Println("updating")
				c.updateUp <- false
				fmt.Println("closing")
				c.Close() // this can take a while but that's ok. this conn won't be used anymore
				// TODO: should add function that returns unflushed data, for dest to query so it can spool it
				return
			} else {
				c.dest.numOut.Add(1)
			}
		case <-tickerFlush.C:
			c.buffered.Flush()
		case <-c.flush:
			c.flushErr <- c.buffered.Flush()
		case <-c.shutdown:
			return
		}
	}
}

func (c *Conn) Write(buf []byte) (int, error) {
	return c.buffered.Write(buf)
}

func (c *Conn) Flush() error {
	log.Println("going to flush mah buffer")
	c.flush <- true
	log.Println("flushing mah buffer, getting error maybe")
	err := <-c.flushErr
	log.Println("flush done; err:", err)
	return err
}

func (c *Conn) Close() error {
	log.Println("Close() called.  sending shutdown")
	c.shutdown <- true
	log.Println("conn close")
	a := c.conn.Close()
	log.Println("closed")
	return a
}
