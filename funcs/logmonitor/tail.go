/*
 *  Copyright (c) 2015 KingSoft.com, Inc. All Rights Reserved
 *  @file: tail2.go
 *  @brief:
 *  @author: suxiaolin(suxiaolin@kingsoft.com)
 *  @date: 2016/01/13 15:03:15
 *  @version: 0.0.1
 *  @history:
 */

package logmonitor

import (
	"bufio"
	//"fmt"
	"io"
	"os/exec"
)

// 通过tail -f命令读取文件最新行，最新行会被monitor保存到每个rule的解析器，实现每个rule根据自己的keywords判断
type Tail struct {
	// 通道，缓存读取到的行数，最大缓存LINE_BUFFER_SIZE行
	Line chan *string
	// 文件名
	fileName string
	// cmd持续运行tail -f
	cmd *exec.Cmd
	// 保存cmd的stdout
	stdOut io.ReadCloser
	// 将stdOut转为reader，方便按行解析
	reader  *bufio.Reader
	running bool
}

// 通过tail -f命令读取文件最新行，然后解析新行
func NewTail(fileName string) (tail *Tail, err error) {
	tail = &Tail{}
	tail.Line = make(chan *string, LINE_BUFFER_SIZE)
	tail.running = true
	tail.fileName = fileName
	// 使用的tail -F，同tail -f
	tail.cmd = exec.Command("/usr/bin/tail", "-F", tail.fileName)
	tail.stdOut, err = tail.cmd.StdoutPipe()
	if err != nil {
		return
	}
	tail.cmd.Start()
	tail.cmd.Run()
	tail.reader = bufio.NewReader(tail.stdOut)
	// 持续运行并解析
	go tail.tail()
	return
}

// 解析读取到的每一行，放入到
func (this *Tail) tail() {
	defer func() {
		if e := recover(); e != nil {
			//catch a panic
		}
		//fmt.Println("tail done")
	}()
	var lineBytes []byte
	var lineSize int
	var err error
	for this.running {
		lineBytes, err = this.reader.ReadBytes('\n')
		if err != nil {
			continue
		}
		lineSize = len(lineBytes)
		if lineSize == 0 {
			continue
		}
		/* delete \n from line */
		line := string(lineBytes[0 : lineSize-1])
		this.Line <- &line
	}
}

func (this *Tail) Close() {
	//defer func() {
	//	fmt.Println("close")
	//}()
	this.running = false
	this.stdOut.Close()
	this.cmd.Process.Kill()
	this.cmd.Wait()
	close(this.Line)
}
