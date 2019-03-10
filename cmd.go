package main

//func main() {
//	var out bytes.Buffer
//	pwd,err := filepath.Abs("./")
//	if err != nil{
//		panic (err)
//	}
//	localPath := path.Join(pwd,"httpbin/spec.json")
//	awsPath := "s3://mybucket/" + "httpbin/spec.json"
//	fmt.Println(awsPath)
//	cmd := exec.Command("aws",
//		"s3",
//		"--endpoint=http://localhost:9000/",
//		"cp",
//		localPath,
//		awsPath)
//	cmd.Stdout = &out
//	res := cmd.Run()
//	fmt.Println(out.String())
//
//	if res != nil{
//		panic (res)
//	}
//}
