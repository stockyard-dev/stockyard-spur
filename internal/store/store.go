package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Hook struct{
	ID string `json:"id"`
	Name string `json:"name"`
	URL string `json:"url"`
	Event string `json:"event"`
	Secret string `json:"secret"`
	Active string `json:"active"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"spur.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS hooks(id TEXT PRIMARY KEY,name TEXT NOT NULL,url TEXT DEFAULT '',event TEXT DEFAULT '',secret TEXT DEFAULT '',active TEXT DEFAULT 'true',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Hook)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO hooks(id,name,url,event,secret,active,created_at)VALUES(?,?,?,?,?,?,?)`,e.ID,e.Name,e.URL,e.Event,e.Secret,e.Active,e.CreatedAt);return err}
func(d *DB)Get(id string)*Hook{var e Hook;if d.db.QueryRow(`SELECT id,name,url,event,secret,active,created_at FROM hooks WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.URL,&e.Event,&e.Secret,&e.Active,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Hook{rows,_:=d.db.Query(`SELECT id,name,url,event,secret,active,created_at FROM hooks ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Hook;for rows.Next(){var e Hook;rows.Scan(&e.ID,&e.Name,&e.URL,&e.Event,&e.Secret,&e.Active,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM hooks WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM hooks`).Scan(&n);return n}
