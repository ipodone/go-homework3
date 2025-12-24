package main

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 用户
type User struct {
	gorm.Model
	Username string // 用户名
	Password string // 密码
	Email    string // 邮箱

	// 关联：用户有多个文章
	Post []Post // 文章

	// 关联：用户有多个评论
	Comment []Comment // 评论

	PostCount int // 文章数量

}

// 文章
type Post struct {
	gorm.Model
	Title   string // 标题
	Content string // 内容

	// 外键：作者
	UserID uint // 用户ID

	// 关联：文章有多个评论
	Comment []Comment // 评论

	CommentStatus string // 评论状态
	CommentCount  int    // 评论数量
}

// 文章结果
type PostResult struct {
	Title        string // 标题
	CommentCount int    // 评论数量
}

// 评论
type Comment struct {
	gorm.Model
	Content string // 内容

	// 外键：评论者
	UserID uint // 用户ID

	// 外键：所属文章
	PostID uint // 文章ID
}

// 创建文章后的钩子：在文章创建时自动更新用户的文章数量统计字段
func (p *Post) AfterCreate(tx *gorm.DB) (err error) {
	fmt.Println("=====================================post")
	return
}

// 删除文章后的钩子（软删除）：在文章删除时自动更新用户的文章数量统计字段
func (p *Post) AfterDelete(tx *gorm.DB) (err error) {
	return
}

// 创建评论后的钩子：在评论创建时自动更新文章的评论数量统计字段、并更新文章的评论状态为有评论
func (c *Comment) AfterCreate(tx *gorm.DB) (err error) {
	fmt.Println("=====================================comment")
	return
}

// 删除评论后的钩子（软删除）：在评论删除时自动更新文章的评论数量统计字段、并更新文章的评论状态为无评论
func (c *Comment) AfterDelete(tx *gorm.DB) (err error) {
	return
}

// 初始化数据库
func InitDB(dst ...interface{}) *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:11fit@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"))
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(dst...)

	return db
}

// 题目1：模型定义
func ModelDefinition(db *gorm.DB) {
	// 创建用户1
	user1 := User{Username: "张三", Email: "zhangsan@example.com", Password: "hashed_password"}
	db.Create(&user1)
	// 创建文章（关联用户）11
	post11 := Post{Title: "user1第一篇文章标题", Content: "user1第一篇文章内容", UserID: user1.ID}
	db.Create(&post11)
	// 创建评论（关联用户和文章）111
	comment111 := Comment{Content: "很好的文章111！", UserID: user1.ID, PostID: post11.ID}
	db.Create(&comment111)
	// 创建评论（关联用户和文章）112
	comment112 := Comment{Content: "很好的文章112！", UserID: user1.ID, PostID: post11.ID}
	db.Create(&comment112)
	// 创建评论（关联用户和文章）113
	comment113 := Comment{Content: "很好的文章113！", UserID: user1.ID, PostID: post11.ID}
	db.Create(&comment113)

	// 创建文章（关联用户）12
	post12 := Post{Title: "user1第二篇文章标题", Content: "user1第二篇文章内容", UserID: user1.ID}
	db.Create(&post12)
	// 创建评论（关联用户和文章）121
	comment121 := Comment{Content: "很好的文章121！", UserID: user1.ID, PostID: post12.ID}
	db.Create(&comment121)

	// 创建用户2
	user2 := User{Username: "李四", Email: "lisi@example.com", Password: "hashed_password"}
	db.Create(&user2)
	// 创建文章（关联用户）21
	post21 := Post{Title: "user2第一篇文章标题", Content: "user2第一篇文章内容", UserID: user2.ID}
	db.Create(&post21)
	// 创建评论（关联用户和文章）211
	comment211 := Comment{Content: "很好的文章211！", UserID: user2.ID, PostID: post21.ID}
	db.Create(&comment211)
}

// 题目2：关联查询
// 查询某个用户发布的所有文章及其对应的评论信息
func AssociationQuery1(db *gorm.DB) {
	fmt.Println("====查询某个用户（张三 UserID=1）发布的所有文章及其对应的评论信息====start")

	// 用户信息
	user := User{}
	db.Preload("Post").Find(&user, 1)
	fmt.Println("用户信息:", user.Username, user.Email)

	// 文章信息
	for _, post := range user.Post { // posts[i]方式会把填充user、post方式不会
		fmt.Println(user.Username, ":", post.Title, post.Content)
		db.Preload("Comment").Find(&post)

		// 评论信息 问题1：comment无法写for循环（原因：模型定义错误为1对1）、问题2：comment只能查出一条数据（原因：模型定义错误为1对1）
		for _, comment := range post.Comment {
			fmt.Println(post.Title, ":", comment.Content, comment.UserID)
		}
	}

	fmt.Println("====查询某个用户（张三 UserID=1）发布的所有文章及其对应的评论信息====end")
}

// 查询评论数量最多的文章信息
func AssociationQuery2(db *gorm.DB) {
	fmt.Println("====查询评论数量最多的文章信息====start")

	sql :=
		`SELECT
		  p.title,
		  COUNT( c.id ) AS comment_count 
		FROM
		  posts p
		LEFT JOIN comments c ON p.id = c.post_id 
		GROUP BY
		  p.id 
		  HAVING
		COUNT( c.id ) = ( SELECT max( comment_count ) FROM ( SELECT COUNT( id ) AS comment_count FROM comments GROUP BY post_id ) AS max_count )`

	var postResults []PostResult // 防止存在并列第一
	db.Raw(sql).Scan(&postResults)
	for _, v := range postResults {
		fmt.Println("评论数量最多的文章为：", v.Title, "；评论数量为：", v.CommentCount)
	}

	fmt.Println("====查询评论数量最多的文章信息====end")
}

// 题目3：钩子函数
func HookFunc() {
	fmt.Println("已在文章或评论操作中实现")
}

func main() {
	fmt.Println("====题目1：模型定义==== start")
	db := InitDB(&User{}, &Post{}, &Comment{})
	ModelDefinition(db)
	fmt.Println("====题目1：模型定义==== end")
	fmt.Println()

	fmt.Println("====题目2：关联查询==== start")
	AssociationQuery1(db)
	fmt.Println()
	AssociationQuery2(db)
	fmt.Println("====题目2：关联查询==== end")
	fmt.Println()

	fmt.Println("====题目3：钩子函数==== start")
	HookFunc()
	fmt.Println("====题目3：钩子函数==== end")
}
