package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Hook struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Event string `json:"event"`
	TargetURL string `json:"target_url"`
	Secret string `json:"secret"`
	Enabled int `json:"enabled"`
	DeliveryCount int `json:"delivery_count"`
	FailCount int `json:"fail_count"`
	Status string `json:"status"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"spur.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS hooks(id TEXT PRIMARY KEY,name TEXT NOT NULL,event TEXT DEFAULT '',target_url TEXT DEFAULT '',secret TEXT DEFAULT '',enabled INTEGER DEFAULT 1,delivery_count INTEGER DEFAULT 0,fail_count INTEGER DEFAULT 0,status TEXT DEFAULT 'active',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Hook)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO hooks(id,name,event,target_url,secret,enabled,delivery_count,fail_count,status,created_at)VALUES(?,?,?,?,?,?,?,?,?,?)`,e.ID,e.Name,e.Event,e.TargetURL,e.Secret,e.Enabled,e.DeliveryCount,e.FailCount,e.Status,e.CreatedAt);return err}
func(d *DB)Get(id string)*Hook{var e Hook;if d.db.QueryRow(`SELECT id,name,event,target_url,secret,enabled,delivery_count,fail_count,status,created_at FROM hooks WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.Event,&e.TargetURL,&e.Secret,&e.Enabled,&e.DeliveryCount,&e.FailCount,&e.Status,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Hook{rows,_:=d.db.Query(`SELECT id,name,event,target_url,secret,enabled,delivery_count,fail_count,status,created_at FROM hooks ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Hook;for rows.Next(){var e Hook;rows.Scan(&e.ID,&e.Name,&e.Event,&e.TargetURL,&e.Secret,&e.Enabled,&e.DeliveryCount,&e.FailCount,&e.Status,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Update(e *Hook)error{_,err:=d.db.Exec(`UPDATE hooks SET name=?,event=?,target_url=?,secret=?,enabled=?,delivery_count=?,fail_count=?,status=? WHERE id=?`,e.Name,e.Event,e.TargetURL,e.Secret,e.Enabled,e.DeliveryCount,e.FailCount,e.Status,e.ID);return err}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM hooks WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM hooks`).Scan(&n);return n}

func(d *DB)Search(q string, filters map[string]string)[]Hook{
    where:="1=1"
    args:=[]any{}
    if q!=""{
        where+=" AND (name LIKE ?)"
        args=append(args,"%"+q+"%");
    }
    if v,ok:=filters["enabled"];ok&&v!=""{where+=" AND enabled=?";args=append(args,v)}
    if v,ok:=filters["status"];ok&&v!=""{where+=" AND status=?";args=append(args,v)}
    rows,_:=d.db.Query(`SELECT id,name,event,target_url,secret,enabled,delivery_count,fail_count,status,created_at FROM hooks WHERE `+where+` ORDER BY created_at DESC`,args...)
    if rows==nil{return nil};defer rows.Close()
    var o []Hook;for rows.Next(){var e Hook;rows.Scan(&e.ID,&e.Name,&e.Event,&e.TargetURL,&e.Secret,&e.Enabled,&e.DeliveryCount,&e.FailCount,&e.Status,&e.CreatedAt);o=append(o,e)};return o
}

func(d *DB)Stats()map[string]any{
    m:=map[string]any{"total":d.Count()}
    rows,_:=d.db.Query(`SELECT status,COUNT(*) FROM hooks GROUP BY status`)
    if rows!=nil{defer rows.Close();by:=map[string]int{};for rows.Next(){var s string;var c int;rows.Scan(&s,&c);by[s]=c};m["by_status"]=by}
    return m
}
