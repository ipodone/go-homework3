package main

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 用户
type User struct {
	gorm.Model
	Username string `gorm:"type:varchar(100);comment:用户名"` // 用户名
	Password string `gorm:"type:varchar(100);comment:密码"`  // 密码
	Email    string `gorm:"type:varchar(100);comment:邮箱"`  // 邮箱

	// 关联：用户有多个文章
	Post []Post // 文章

	// 关联：用户有多个评论
	Comment []Comment // 评论

	PostCount uint // 文章数量
}

// 文章
type Post struct {
	gorm.Model
	Title   string `gorm:"type:varchar(100);comment:标题"` // 标题
	Content string `gorm:"type:varchar(100);comment:内容"` // 内容

	// 外键：作者
	UserID uint // 用户ID

	// 关联：文章有多个评论
	Comment []Comment // 评论

	CommentStatus string `gorm:"type:varchar(100);comment:评论状态"` // 评论状态（""-从来没有评论、有评论-有评论、无评论-有过但删完了）
	CommentCount  uint   // 评论数量
}

// 文章结果
type PostResult struct {
	Title        string // 标题
	CommentCount uint   // 评论数量
}

// 评论
type Comment struct {
	gorm.Model
	Content string `gorm:"type:varchar(100);comment:内容"` // 内容

	// 外键：评论者
	UserID uint // 用户ID

	// 外键：所属文章
	PostID uint // 文章ID
}

// 创建文章后的钩子：在文章创建时自动更新用户的文章数量统计字段
func (p *Post) AfterCreate(tx *gorm.DB) (err error) {
	// 根本不用查询
	// tx.Find(&user, p.UserID) // tx.Debug().Find(&user) 它查出来所有数据，但接收只有一个对象，所有只有一条数据
	// 这里使用数据库表字段名称：且Update时，需要使用Model，构建SQL里的表名
	tx.Model(&User{}).Where("id = ?", p.UserID).Update("post_count", gorm.Expr("post_count + 1")) // 这里要用表达式，不能++1
	fmt.Println("在文章创建时自动更新用户的文章数量统计字段 完成")
	return
}

// 删除文章后的钩子（软删除）：在文章删除时自动更新用户的文章数量统计字段
func (p *Post) AfterDelete(tx *gorm.DB) (err error) {
	tx.Model(&User{}).Where("id = ? and post_count > 0", p.UserID).Update("post_count", gorm.Expr("post_count - 1"))
	fmt.Println("在文章删除时自动更新用户的文章数量统计字段 完成")
	return
}

// 创建评论后的钩子：在评论创建时自动更新文章的评论数量统计字段、并更新文章的评论状态为有评论
func (c *Comment) AfterCreate(tx *gorm.DB) (err error) {
	post := Post{}
	tx.Find(&post, c.PostID)

	if post.CommentStatus == "有评论" {
		tx.Model(&Post{}).Where("id = ?", c.PostID).Update("comment_count", gorm.Expr("comment_count + 1"))
		fmt.Println("在评论创建时自动更新文章的评论数量统计字段、并更新文章的评论状态为有评论 完成")
		return
	}

	tx.Model(&Post{}).Where("id = ?", c.PostID).
		Updates(map[string]interface{}{"comment_status": "有评论", "comment_count": 1})
	fmt.Println("在评论创建时自动更新文章的评论数量统计字段、并更新文章的评论状态为有评论 完成")
	return
}

// Go 中的 GORM 操作和钩子都是同步的，如果需要异步处理，应该明确地使用 goroutine 或任务队列，并注意数据一致性和连接管理。
// 在事务中，钩子也是同步的，且与事务同步（即事务对钩子有效）
// err := db.Transaction(func(tx *gorm.DB) error {
//     user := User{Name: "bob"}

//     // BeforeCreate 在事务内执行
//     if err := tx.Create(&user).Error; err != nil {
//         return err
//     }

//	    // AfterCreate 也在事务内执行
//	    // 如果这里出错，整个事务回滚
//	    return nil
//	})
//
// 删除评论前的钩子（软删除）：在评论删除时自动更新文章的评论数量统计字段、并更新文章的评论状态为无评论
func (c *Comment) AfterDelete(tx *gorm.DB) (err error) {
	post := Post{}
	tx.Find(&post, c.PostID)

	if post.CommentStatus == "无评论" || post.CommentStatus == "" {
		return
	}

	// 此处的查询返回为0行，原因：因为相关数据已被逻辑删除，即使断点到这里在navicat中通过sql能查出数据
	// 1、因为使用AfterDelete：此时数据已记录到数据库中，尚未提交事务；
	// 2、使用BeforeDelete可以：此时数据未记录到数据库中
	// 3、使用AfterDelete+Unscoped可以：你可以使用Unscoped来查询到被软删除的记录
	// comments := []Comment{}
	// tx.Debug().Where("post_id = ?", c.PostID).Find(&comments)
	// tx.Debug().Unscoped().Where("post_id = ?", c.PostID).Find(&comments)

	// 1、创建、查询、更新、删除都有返回值（注意：返回的错误、影响行数判断<此处为警告>）
	// 2、但创建、查询会返回数据（自动回填数据到结构体）；更新、删除不会
	// 3、！！！创建时，created_at、updated_at更新；更新时，updated_at更新；删除时，deleted_at更新
	// 4、！！！写原生SQL时注意：（1）上面3个时间的更新 （2）是否查有效数据，即deleted_at is null
	// for _, v := range comments {      // 删除一条评论时会查出多条处理，这里不对
	// 	tx.Model(&Post{}).Where("id = ?", v.PostID).Update("comment_count", gorm.Expr("comment_count - 1"))
	// 	if post.CommentCount == 1 == 1 {
	// 		tx.Model(&Post{}).Where("id = ?", v.PostID).Update("comment_status", "无评论")
	// 	}
	// 	fmt.Println("在评论删除时自动更新文章的评论数量统计字段、并更新文章的评论状态为无评论 完成")
	// }

	tx.Debug().Model(&Post{}).Where("id = ?", post.ID).Update("comment_count", gorm.Expr("comment_count - 1"))
	if post.CommentCount == 1 {
		tx.Model(&Post{}).Where("id = ?", post.ID).Update("comment_status", "无评论")
	}
	fmt.Println("在评论删除时自动更新文章的评论数量统计字段、并更新文章的评论状态为无评论 完成")

	return
}

// 初始化数据库
func InitDB(dst ...interface{}) *gorm.DB {
	// db, err := gorm.Open(mysql.Open("root:11fit@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"))
	dsn := "root:11fit@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 禁止创建物理外键，使用逻辑外键
	})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(dst...)

	return db
}

// 题目1：模型定义
func ModelDefinition(db *gorm.DB) {
	aaa := 2
	if aaa == 1 { // 测试使用
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
	} else if aaa == 2 {
		// 删除文章1（同时更新用户文章数量）及评论（同时更新文章评论状态及评论数量）
		// 删除文章分3步：1、更新文章评论状态及评论数量 2、删除评论 3、删除文章
		post := Post{}
		post.ID = 1
		post.UserID = 1
		comment := Comment{}
		comment.PostID = post.ID

		// 此处能查出来评论列表
		// comments := []Comment{}
		// db.Debug().Where("post_id = ?", 1).Find(&comments)

		// 这种删除文章时，更新评论状态及评论数量的操作：
		// 1、放在删除评论的钩子里不合适（原因：影响单条评论删除）
		// 2、放在删除文章的钩子里不合适（原因：与删除评论钩子里的逻辑重复）
		// 3、不行放在钩子外面吧（即这里的删除评论前或后）
		db.Debug().Transaction(func(tx *gorm.DB) error {
			db.Model(&Post{}).Where("id = ?", post.ID).
				Updates(map[string]interface{}{"comment_status": "无评论", "comment_count": 0})

			if err := tx.Where("post_id = ?", post.ID).Delete(&comment).Error; err != nil {
				return err
			}

			result := db.Find(&post, post.ID)
			if result.RowsAffected > 0 {
				if err := tx.Delete(&post, post.ID).Error; err != nil {
					return err
				}
			}

			return nil
		})

	} else if aaa == 3 {
		// 删除评论3
		comment := Comment{PostID: 1}
		comment.ID = 3
		result := db.Find(&comment, comment.ID)
		if result.RowsAffected > 0 {
			db.Delete(&comment, comment.ID)
		}
	}
}

// 题目2：关联查询
// 查询某个用户发布的所有文章及其对应的评论信息
func AssociationQuery1(db *gorm.DB) {
	fmt.Println("====查询某个用户（张三 UserID=1）发布的所有文章及其对应的评论信息====start")

	// 用户信息
	user := User{}
	db.Preload("Post").Find(&user, 1) // 这里使用对象名称
	fmt.Println("用户信息:", user.Username, user.Email)

	// 文章信息
	for _, post := range user.Post { // posts[i]方式会填充user、post方式不会
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
		  AND p.deleted_at IS NULL 
		GROUP BY
		  p.id 
		  HAVING
		COUNT( c.id ) = ( SELECT max( comment_count ) FROM ( SELECT COUNT( id ) AS comment_count FROM comments WHERE deleted_at IS NULL GROUP BY post_id ) AS max_count )`

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
