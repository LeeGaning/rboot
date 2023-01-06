package wechat

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fastjson"
)

//微信推送相关数据
type VxPush struct {
	Appid       string
	Secret      string
	Template_id string
	AccessToken Access_token
	Url         string
	DB_Url      string
	db          *sqlx.DB
}

//输出消息体
type PushMsg struct {
	msgID    string //消息id
	NameId   string //当前消息的用户名
	MsgTime  string //消息的时间
	MsgTitle string //消息title
	Msg      string //消息内容
	Remark   string //消息备注
	Url      string
	UserName string
	wechatId string //当前用户的vxid
}

//access_token的json结构体
type Access_token struct {
	Token      string `json:"access_token"`
	Expires_in int64  `json:"expires_in"`
	Expires_at time.Time
}

func (t Access_token) IsTokenExpired() bool {
	if len(t.Token) < 1 {
		return true
	}
	return time.Now().After(t.Expires_at)
}

type User struct {
	Id       int    `db:"userID"`
	Phone    string `db:"phone"`
	Name     string `db:"name"`
	WechatId string `db:"wechat_number"`
}

// CheckMobile 检验手机号
func CheckMobile(phone string) bool {
	// 匹配规则
	// ^1第一位为一
	// [345789]{1} 后接一位345789 的数字
	// \\d \d的转义 表示数字 {9} 接9位
	// $ 结束符
	regRuler := "^1[345789]{1}\\d{9}$"

	// 正则调用规则
	reg := regexp.MustCompile(regRuler)

	// 返回 MatchString 是否匹配
	return reg.MatchString(phone)

}

//此结构体为vx推送的json结构体
type VxPushJson struct {
}

var mapUser map[string]string = map[string]string{"lee": "oPylu6WhSKTMrbL8JBoENzbF1AsQ", "cb_lee": "oZuda5g2FMWEnlOq3pbiWuBQHaV8"}

func (v *VxPush) GetAccessToken() {
	GetUrl := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" + url.QueryEscape(v.Appid) + "&secret=" + url.QueryEscape(v.Secret)
	resp, err := http.Get(GetUrl)
	if err != nil {
		fmt.Println("GetAccessToken失败")
		panic(err.Error())
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Println("GetAccessToken请求失败")
		panic(err.Error())
	}

	var access_token Access_token
	// fmt.Println(string(body))
	err_json := json.Unmarshal(body, &access_token)
	if err_json != nil {
		fmt.Println("Access_token解析失败")
		panic(err.Error())
	}
	if access_token.Expires_in < 300 {
		panic("Access_token解析失败")
	}
	access_token.Expires_at = time.Now().Add(time.Second * time.Duration(access_token.Expires_in-30))
	// fmt.Println(access_token)
	v.AccessToken = access_token
	//每次获取的Access_token有效期限是7200秒，我这里提前五秒刷新获取，防止程序延迟，导致Access_token失效，消息不能及时发送

}

//信息推送函数,推送成功，返回一个字符，TRUEorFALSE,传参传入一个消息体，然后将信息时间和信息用户id组合加密，并推送到微信，然后存储到mysql数据库中
//msgID就为，用户id+当前时间，然后进行md5加密
func (v *VxPush) MsgPush(m *PushMsg) string {
	if v.AccessToken.IsTokenExpired() {
		v.GetAccessToken()
		logrus.Info("flush token", v.AccessToken.Expires_at)
	}
	err := v.MsgID(m)
	if err != nil {
		fmt.Println("获取id失败，请检查是否是注册用户: ", err.Error())
		return "推送失败"
	}
	return "推送成功"
}

// //加密生成Msgid的函数
// func (v *VxPush) MsgID(nameid string, time_now string) string {
// 	str1 := nameid + time_now + string(rand.Intn(9999))
// 	result := md5.Sum([]byte(str1))
// 	msgid := fmt.Sprintf("%x", result)
// 	return msgid
// }

//加密生成Msgid的函数,并且获得当前用户的vxid
func (v *VxPush) MsgID(m *PushMsg) error {
	m.MsgTime = time.Now().Format("2006-01-02 15:04:05")
	str1 := m.NameId + m.MsgTime + string(rune(rand.Intn(9999)))
	result := md5.Sum([]byte(str1))
	msgid := fmt.Sprintf("%x", result)
	m.msgID = msgid
	if uid, ok := mapUser[m.NameId]; ok {
		m.wechatId = uid
		err_push := v.VxPush(m)
		if err_push != nil {
			fmt.Println("消息推送失败: ", err_push)
			return err_push
		}
	} else if CheckMobile(m.NameId) {
		if v.db == nil {
			var err error
			if v.db, err = sqlx.Connect("mysql", v.DB_Url); err != nil {
				logrus.Error("mysql connect: ", err)
				return err
			}
		}
		user := User{}
		error := v.db.Get(&user, "SELECT u.userID, u.phone, u.name, c.wechat_number FROM cubemgdb.sys_user u INNER JOIN cubemgdb.sys_user_contact c ON c.user_id = u.userID WHERE u.phone = ?", m.NameId)
		logrus.Info(user, error)
		if len(user.WechatId) > 0 {
			m.wechatId = user.WechatId
			err_push := v.VxPush(m)
			if err_push != nil {
				fmt.Println("消息推送失败: ", err_push)
				return err_push
			}
		} else {
			logrus.Error("unkown nameid ", m.NameId)
		}
	} else if len(m.UserName) > 0 {
		if v.db == nil {
			var err error
			if v.db, err = sqlx.Connect("mysql", v.DB_Url); err != nil {
				logrus.Error("mysql connect: ", err)
				return err
			}
		}
		user := User{}
		error := v.db.Get(&user, "SELECT u.userID, u.phone, u.name, c.wechat_number FROM cubemgdb.sys_user u INNER JOIN cubemgdb.sys_user_contact c ON c.user_id = u.userID WHERE locate(?,u.name)", m.UserName)
		logrus.Info(user, error)
		if len(user.WechatId) > 0 {
			m.wechatId = user.WechatId
			err_push := v.VxPush(m)
			if err_push != nil {
				fmt.Println("消息推送失败: ", err_push)
				return err_push
			}
		} else {
			logrus.Error("unkown nameid ", m.NameId)
		}
	} else {
		logrus.Error("unkown nameid ", m.NameId)
	}

	return nil
}

//微信推送函数，此函数用于推送微信
func (v *VxPush) VxPush(m *PushMsg) error {
	strTemplate := `{
		"touser": %q,
		"template_id":%q,
		"url": %q,
		"topcolor": "#FF0000",
		"data": {
			"title1": {
				"value": "报警类型",
				"color": "#A8A8A8"
			},
			"title2": {
				"value": "报警设备",
				"color": "#A8A8A8"
			},
			"title3": {
				"value": "报警时间",
				"color": "#A8A8A8"
			},
			"title4": {
				"value": "报警内容",
				"color": "#A8A8A8"
			},
			"first": {
				"value": %q,
				"color": "#A8A8A8"
			},
			"keyword1": {
				"value": "故障"
			},
			"keyword2": {
				"value": "设备1"
			},
			"keyword3": {
				"value": %q
			},
			"keyword4": {
				"value": %q
			},
			"remark": {
				"value": %q
			}
		}
	}`
	if len(m.Url) == 0 {
		m.Url = v.Url + "/msg?msgid=" + m.msgID + "&nameid=" + m.NameId
	}
	if len(m.Remark) == 0 {
		m.Remark = "本次推送由robot自动发送\n"
	}
	JsonVxPush := []byte(fmt.Sprintf(strTemplate, m.wechatId, v.Template_id, m.Url, m.MsgTitle, m.MsgTime, m.Msg, m.Remark))
	url_vx := "https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=" + url.QueryEscape(v.AccessToken.Token)
	req, err := http.NewRequest("POST", url_vx, bytes.NewBuffer(JsonVxPush))
	if err != nil {
		fmt.Println("推送请求包构造失败！: ", err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("推送请求包请求失败！: ", err.Error())
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("请求内容body，读取失败", err.Error())
		return err
	}
	logrus.Debug(string(JsonVxPush), string(body))
	if fastjson.GetInt(body, "errcode") != 0 {
		logrus.Error("消息发送失败", fastjson.GetString(body, "errmsg"))
	} else {
		logrus.Info("消息发送成功", fastjson.GetString(body, "msgid"))

	}

	//消息推送成功后，现在把消息存入mysql中
	// err_sql := v.ReadMsgPush(m)
	// if err_sql != nil {
	// 	fmt.Println("消息写入数据库失败: ", err_sql)
	// 	return err_sql
	// }
	return nil
}

// //将信息存入mysql表单中
// func (v *VxPush) ReadMsgPush(m *PushMsg) error {
// 	sqlStr := "INSERT INTO msgpush (msgid,nameid,msgtitle,msgcontent,msgtime) VALUES(?,?,?,?,?);"
// 	stmr, err := Db.Prepare(sqlStr)
// 	if err != nil {
// 		fmt.Println("预编译出现异常: ", err)
// 		return err
// 	}
// 	_, err_sql := stmr.Exec(m.MsgID, m.NameId, m.MsgTitle, m.Msg, m.MsgTime)
// 	if err_sql != nil {
// 		fmt.Println("sql执行异常: ", err_sql)
// 		return err_sql
// 	}
// 	return nil
// }

// //查询当前用户的msgid的信息
// func SelecMsg(msgid string, nameid string) (*PushMsg, error) {
// 	var msg PushMsg
// 	sqlStr := "SELECT msgid,nameid,msgtitle,msgcontent,msgtime FROM msgpush WHERE msgid =? AND nameid = ?;"
// 	row := Db.QueryRow(sqlStr, msgid, nameid)
// 	err := row.Scan(&msg.MsgID, &msg.NameId, &msg.MsgTitle, &msg.Msg, &msg.MsgTime)
// 	if err != nil {
// 		fmt.Println("数据查询失败: ", err)
// 		return nil, err
// 	}
// 	return &msg, nil
// }

//用于初始化创建VxPush结构体
func NewVxPush() *VxPush {
	return &VxPush{
		// 初始化微信
		Appid:       os.Getenv("WX_APPID"),
		Secret:      os.Getenv("WX_SECRET"),
		Template_id: os.Getenv("WX_TEMPLATEID"),
		Url:         os.Getenv("WX_URL"),
		DB_Url:      os.Getenv("DB_URL"),
	}
}
