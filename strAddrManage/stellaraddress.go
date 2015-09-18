package strAddrManage

import (
	"ExchangeZTC/models/databasefunction"
	"ExchangeZTC/models/stellarwebsocket"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"os"
	"strconv"
)

type StellarAddressDefine struct {
	PublicAddress string `json:"public"`
	PrivateSeed   string `json:"private"`
}

type StellarAddressManager struct {
	isFromFile   bool
	confFilePath string
	logFilePath  string
	StrAddrs     []*StellarAddressDefine
	dbPointer    orm.Ormer
}

type StellarAddrInterface interface {
	Init(fromFile bool, accCount int)
	UpdateDB() (int, error)
	Destroy()
}

func (this *StellarAddressManager) Init(fromFile bool, accCount int) {
	this.isFromFile = fromFile
	this.StrAddrs = make([]*StellarAddressDefine, 0)

	basePath := "E:/Work/Code/Go/src/UpdateStellarAddrDB/"

	this.confFilePath = basePath + "confg.cof"
	this.logFilePath = basePath + "create.log"

	this.regDatabase()

	var err error
	if this.isFromFile {
		err = this.getDatasFromFile(this.confFilePath)
	} else {
		err = this.createDatas(accCount, this.logFilePath)
	}

	if err != nil {
		fmt.Printf("[StellarAddressManager:Init]\r\n\tError = [%s]\r\n", err.Error())
	}
}

func (this *StellarAddressManager) UpdateDB() (int64, error) {
	if this.dbPointer != nil {
		ret := this.copyFrom(this.StrAddrs)
		successNums, err := this.dbPointer.InsertMulti(1, ret)
		return successNums, err
	}
	return 0, errors.New("Database connect is not exist!")
}

func (this *StellarAddressManager) Destroy() {
	this.dbPointer = nil
	this.StrAddrs = nil
}

func (this *StellarAddressManager) copyFrom(src []*StellarAddressDefine) (ret []databasefunction.StellarAddrT) {
	length := len(src)
	if length > 0 {
		ret = make([]databasefunction.StellarAddrT, length)
		for i := 0; i < length; i++ {
			b, err := this.checkBalance(src[i].PublicAddress)
			if b == 0 && err == nil {
				ret[i] = databasefunction.StellarAddrT{
					StrAddr:  src[i].PublicAddress,
					CheckOut: 0,
					EStatus:  -1,
				}
			} else if err != nil {
				fmt.Println("[ StellarBalanceListener:checkBalance() ]", err)

			} else {
				fmt.Printf("[ StellarBalanceListener:checkBalance() ]\r\n\t address = %s \t balance = %d\r\n",
					src[i].PublicAddress, b)
			}
		}
	}
	return
}

func (this *StellarAddressManager) getDatasFromFile(filePath string) (err error) {
	finfo, err := os.Stat(filePath)
	if err == nil {
		if finfo.IsDir() == true {
			err := errors.New(filePath + " is not a file!")
			panic(err)
		} else {
			datas, err := ioutil.ReadFile(filePath)
			if err == nil {
				err = json.Unmarshal(datas, &this.StrAddrs)
			}
		}
	}
	return err
}

func (this *StellarAddressManager) createDatas(accCount int, createFilePath string) error {
	if stellarwebsocket.StellarWSInstance == nil {
		stellarwebsocket.InitStellarWSInstance()
	}

	this.StrAddrs = make([]*StellarAddressDefine, accCount)

	f, err := os.OpenFile(createFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeType)
	defer f.Close()
	if err != nil {
		panic(err.Error())
	}

	for i := 0; i < accCount; i++ {
		retMap, err := stellarwebsocket.StellarWSInstance.CreateUser()

		if err == nil {
			result, ok := retMap["result"]
			if ok {
				status, ok := result.(map[string]interface{})["status"]
				if ok {
					if status.(string) == "success" {
						// 创建账号成功
						account_id, _ := result.(map[string]interface{})["account_id"]
						master_seed, _ := result.(map[string]interface{})["master_seed"]
						output := fmt.Sprintf("[ StellarAddressManager:createDatas() ]\r\n\taccount_id = %s ; master_seed = %s\r\n\r\n",
							account_id.(string), master_seed.(string))
						fmt.Printf(output)
						_, err = f.WriteString(output)
						if err != nil {
							return err
						}
						this.StrAddrs[i] = &StellarAddressDefine{
							PublicAddress: account_id.(string),
							PrivateSeed:   master_seed.(string),
						}
					}
				} else {
					i--
				}
			} else {
				i--
			}
		} else {
			i--
		}
	}
	return nil
}

func (this *StellarAddressManager) regDatabase() {
	orm.RegisterModel(new(databasefunction.StellarAddrT))
	orm.Debug = true
	orm.RegisterDriver("mysql", orm.DR_MySQL)

	dataSrcName := "root:1234@tcp(localhost:3306)/default?charset=utf8&loc=Asia%2FShanghai"
	orm.RegisterDataBase("default", "mysql", dataSrcName, 30)

	dataSrcName = "root:1234@tcp(localhost:3306)/stellaraddr_info?charset=utf8&loc=Asia%2FShanghai"
	orm.RegisterDataBase("stellaraddr_info", "mysql", dataSrcName, 30)
	this.dbPointer = orm.NewOrm()
	this.dbPointer.Using("stellaraddr_info")

	orm.RunSyncdb("stellaraddr_info", false, true)
}

func (this *StellarAddressManager) checkBalance(addr string) (uint64, error) {
	if stellarwebsocket.StellarWSInstance == nil {
		stellarwebsocket.InitStellarWSInstance()
	}
	retMap, err := stellarwebsocket.StellarWSInstance.GetBalance(addr)

	// fmt.Println("[ StellarBalanceListener:checkBalance() ] = ", retMap)

	if err == nil {
		result, ok := retMap["result"]
		if ok {
			status, ok := result.(map[string]interface{})["status"]
			if ok {
				if status.(string) == "success" { //账号已经存在
					account_data, _ := result.(map[string]interface{})["account_data"]
					balance, _ := account_data.(map[string]interface{})["Balance"]
					b, err := strconv.ParseUint(balance.(string), 10, 64)
					return b, err
				} else {
					error_message, ok := result.(map[string]interface{})["error_message"]
					if ok && error_message.(string) == "Account not found." {
						return 0, nil
					} else {
						err = errors.New("Listen to " + addr + " has error -> Read undefined error message :" + error_message.(string))
					}
				}
			} else {
				err = errors.New("Listen to " + addr + " has error -> Can not get 'result->status' form response body.")
			}
		} else {
			err = errors.New("Listen to " + addr + " has error -> Can not get 'result' form response body.")
		}
	}
	fmt.Println("[ StellarBalanceListener:checkBalance() ] = ", err)
	return 0, err
}
