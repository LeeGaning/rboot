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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fastjson"
)

//微信推送相关数据
type VxPush struct {
	Appid       string
	Secret      string
	Template_id string
	AccessToken string
	Url         string
}

//输出消息体
type PushMsg struct {
	MsgID    string //消息id
	NameId   string //当前消息的用户名
	MsgTime  string //消息的时间
	MsgTitle string //消息title
	Msg      string //消息内容
	Userid   string //当前用户的vxid
}

//access_token的json结构体
type Access_token struct {
	Access_token string `json:"access_token"`
	Expires_in   int    `json:"expires_in"`
}

//此结构体为vx推送的json结构体
type VxPushJson struct {
}

var mapUser map[string]string = map[string]string{"lee": "oPylu6WhSKTMrbL8JBoENzbF1AsQ", "cb_lee": "oZuda5g2FMWEnlOq3pbiWuBQHaV8"}

func (v *VxPush) GetAccessToken() {
	for {
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
		err_json := json.Unmarshal(body, &access_token)
		if err_json != nil {
			fmt.Println("Access_token解析失败")
			panic(err.Error())
		}
		if access_token.Expires_in != 7200 {
			panic("Access_token解析失败")
		}
		fmt.Println(access_token.Access_token)
		v.AccessToken = access_token.Access_token
		//每次获取的Access_token有效期限是7200秒，我这里提前五秒刷新获取，防止程序延迟，导致Access_token失效，消息不能及时发送
		time.Sleep(time.Second * 7195)
	}

}

//信息推送函数,推送成功，返回一个字符，TRUEorFALSE,传参传入一个消息体，然后将信息时间和信息用户id组合加密，并推送到微信，然后存储到mysql数据库中
//msgID就为，用户id+当前时间，然后进行md5加密
func (v *VxPush) MsgPush(m *PushMsg) string {
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
	m.MsgID = msgid

	if uid, ok := mapUser[m.NameId]; ok {
		m.Userid = uid
		err_push := v.VxPush(m)
		if err_push != nil {
			fmt.Println("消息推送失败: ", err_push)
			return err_push
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
				"value": %q,
				"color": "#A8A8A8"
			},
			"title2": {
				"value": "通知内容:\t\t\t\t",
				"color": "#A8A8A8"
			},
			"title3": {
				"value": "通知时间:\t\t\t\t",
				"color": "#A8A8A8"
			},
			"title4": {
				"value": "备注:\t\t\t\t",
				"color": "#A8A8A8"
			},
			"content1": {
				"value": %q
			},
			"content2": {
				"value": %q
			},
			"content3": {
				"value": "本次推送由robot自动发送\n"
			}
		}
	}`
	msgUrl := v.Url + "/msg?msgid=" + m.MsgID + "&nameid=" + m.NameId
	JsonVxPush := []byte(fmt.Sprintf(strTemplate, m.Userid, v.Template_id, msgUrl, m.MsgTitle, m.Msg, m.MsgTime))
	url_vx := "https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=" + url.QueryEscape(v.AccessToken)
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
		Url:         os.Getenv("WX_URL")}
}
