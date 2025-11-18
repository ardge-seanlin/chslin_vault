# Rust write!() 宏詳解

本指南詳細說明 Rust 中 `write!()` 宏的工作原理，特別是在你的 `AppError::Display` 實現中的用法。

## 目錄

- [快速概覽](#快速概覽)
- [write!() 宏的三種形式](#write-宏的三種形式)
- [你的代碼詳解](#你的代碼詳解)
- [fmt::Display trait](#fmtdisplay-trait)
- [Formatter 物件](#formatter-物件)
- [實際應用範例](#實際應用範例)
- [與 println!() 的比較](#與-println-的比較)
- [常見錯誤](#常見錯誤)

## 快速概覽

### 你的代碼片段

```rust
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::MdnsError(msg) => write!(f, "mDNS error: {}", msg),
            //                          ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
            //                          這是 write!() 宏
        }
    }
}
```

### 這一行做了什麼事

```
write!(f, "mDNS error: {}", msg)

   ↓

1. 取得 Formatter 物件 f
2. 格式化字符串 "mDNS error: {}"
3. 將變數 msg 插入 {}
4. 將格式化結果寫入 f
5. 返回 Result（成功或失敗）
```

## write!() 宏的三種形式

### 1️⃣ 寫入到 Formatter（你使用的方式）

```rust
use std::fmt::{self, Formatter};

fn format_error(f: &mut Formatter, msg: &str) -> fmt::Result {
    write!(f, "Error: {}", msg)
    // ↑ 寫入到 Formatter（用於 Display/Debug trait）
}
```

**場景：** 實現 `Display` 或 `Debug` trait

### 2️⃣ 寫入到檔案

```rust
use std::fs::File;
use std::io::Write;

let mut file = File::create("output.txt")?;
write!(file, "Hello, {}", name)?;
// ↑ 寫入到檔案
```

**場景：** 檔案 I/O

### 3️⃣ 寫入到字符串

```rust
let mut s = String::new();
write!(s, "Hello, {}", name)?;
// ↑ 寫入到字符串
```

**場景：** 字符串格式化

## 你的代碼詳解

### 完整上下文

```rust
// 這是你的 AppError 定義
#[derive(Debug, Serialize)]
pub enum AppError {
    MdnsError(String),
    InvalidData(String),
    InternalError(String),
}

// 實現 Display trait
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            // 當 AppError 是 MdnsError 時
            AppError::MdnsError(msg) => {
                // 格式化並寫入
                write!(f, "mDNS error: {}", msg)
            }

            AppError::InvalidData(msg) => {
                write!(f, "Invalid data: {}", msg)
            }

            AppError::InternalError(msg) => {
                write!(f, "Internal error: {}", msg)
            }
        }
    }
}
```

### 逐步分解

```
write!(f, "mDNS error: {}", msg)
│       │  │                 │
│       │  │                 └─ 第 3 個參數：要替換到 {} 的值
│       │  └─────────────────── 第 2 個參數：格式化字符串
│       └────────────────────── 第 1 個參數：Formatter 物件
└─────────────────────────────── write! 宏
```

### 執行流程

```
1️⃣ 接收 Formatter
   f: &mut fmt::Formatter<'_>
   ↓
2️⃣ 解析格式化字符串
   "mDNS error: {}"
   ↓
3️⃣ 識別佔位符
   {} ← 這裡需要插入 msg
   ↓
4️⃣ 格式化參數
   msg.to_string() 或 format!("{}", msg)
   ↓
5️⃣ 寫入結果
   f.write_str("mDNS error: some_message")
   ↓
6️⃣ 返回結果
   Ok(()) 或 Err(fmt::Error)
```

## fmt::Display trait

### 什麼是 Display Trait？

`fmt::Display` 是一個 trait，用來定義「如何將值轉換為易於人類閱讀的字符串」。

### 何時觸發 Display 實現？

```rust
let error = AppError::MdnsError("Service not found".to_string());

// 以下任何操作都會觸發 Display 實現
println!("{}", error);           // 直接格式化
format!("{}", error);             // 返回 String
error.to_string();                // 呼叫 Display trait
eprintln!("{}", error);           // 錯誤輸出
```

### 輸出範例

```rust
let error = AppError::MdnsError("Service not found".to_string());

println!("{}", error);
// 輸出：mDNS error: Service not found

error.to_string();
// 返回：String = "mDNS error: Service not found"
```

### 比較：Display vs Debug

```rust
// Display trait - 易於閱讀的格式
// 你實現的就是這個：
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "mDNS error: {}", msg)  // 自訂格式
    }
}

// Debug trait - 詳細的開發者格式
// 派生自 #[derive(Debug)]：
impl fmt::Debug for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_tuple("AppError")
            .field(&self)
            .finish()
    }
}

// 使用差別
println!("{}", error);      // Display 輸出：mDNS error: Service not found
println!("{:?}", error);    // Debug 輸出：AppError(MdnsError("Service not found"))
println!("{:#?}", error);   // Pretty Debug 輸出（多行）
```

## Formatter 物件

### fmt::Formatter 是什麼？

`Formatter` 是一個物件，代表「格式化的目標」。它負責管理：
- 輸出目的地（控制台、檔案、字符串等）
- 格式化選項（寬度、精度、對齐等）
- 格式化狀態

### Formatter 的常用方法

```rust
use std::fmt::{self, Formatter};

fn custom_display(f: &mut Formatter) -> fmt::Result {
    // 寫入文本
    f.write_str("Hello")?;

    // 寫入整個字符串
    f.write_fmt(format_args!("World"))?;

    // 其他方法
    f.pad("padded");           // 填充
    f.write_char('a');         // 寫單個字符

    Ok(())
}
```

### Formatter 的格式化選項

```rust
impl fmt::Display for MyType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // 取得格式化選項
        let width = f.width();        // 最小寬度
        let precision = f.precision(); // 精度
        let align = f.align();        // 對齐方式

        write!(f, "{:width$}", self.value, width = width)
    }
}
```

## 實際應用範例

### 範例 1：簡單的錯誤顯示（你的代碼）

```rust
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::MdnsError(msg) => write!(f, "mDNS error: {}", msg),
            AppError::InvalidData(msg) => write!(f, "Invalid data: {}", msg),
            AppError::InternalError(msg) => write!(f, "Internal error: {}", msg),
        }
    }
}

// 使用
let error = AppError::mdns("Service discovery failed");
println!("{}", error);  // 輸出：mDNS error: Service discovery failed
```

### 範例 2：複雜的格式化

```rust
struct Point {
    x: i32,
    y: i32,
}

impl fmt::Display for Point {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "({}, {})", self.x, self.y)
    }
}

let p = Point { x: 3, y: 4 };
println!("{}", p);          // 輸出：(3, 4)
println!("{:10}", p);       // 輸出：       (3, 4)  （右對齐，寬度 10）
```

### 範例 3：帶顏色的錯誤輸出

```rust
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::MdnsError(msg) => {
                // ANSI 紅色文本
                write!(f, "\x1b[31mERROR: mDNS - {}\x1b[0m", msg)
            }
            AppError::InvalidData(msg) => {
                write!(f, "\x1b[33mWARN: Invalid data - {}\x1b[0m", msg)
            }
            _ => write!(f, "{:?}", self),
        }
    }
}
```

### 範例 4：自訂錯誤訊息結構

```rust
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::MdnsError(msg) => {
                write!(
                    f,
                    "[{}] mDNS Service Discovery Error: {}",
                    chrono::Local::now().format("%Y-%m-%d %H:%M:%S"),
                    msg
                )
            }
            _ => write!(f, "{:?}", self),
        }
    }
}

// 輸出範例
// [2025-01-15 10:30:45] mDNS Service Discovery Error: Service not found
```

## 與 println!() 的比較

### println!() 的運作原理

```rust
// 這個 println! 調用
println!("Error: {}", error);

// 實際上執行以下步驟：
// 1. 呼叫 format!("Error: {}", error)
// 2. 建立一個 Formatter
// 3. 呼叫 error 的 Display::fmt(f) 方法
// 4. 將結果寫入標準輸出
// 5. 新增換行符
```

### 詳細流程

```rust
// 簡化的 println! 實現
macro_rules! println {
    ($($arg:tt)*) => {
        print!("{}\n", format_args!($($arg)*))
    };
}

// 當執行
println!("{}", error);

// Rust 內部執行
let formatted = error.to_string();  // 呼叫 Display trait
std::io::stdout().write_all(formatted.as_bytes())?;
std::io::stdout().write_all(b"\n")?;
```

## 常見錯誤

### 錯誤 1：忘記使用 write!() 的返回值

```rust
// ❌ 錯誤
impl fmt::Display for MyType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Hello")?;  // 有 ?
        write!(f, "World");   // ❌ 缺少 ?，忽視了錯誤
        Ok(())
    }
}

// ✅ 正確
impl fmt::Display for MyType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Hello")?;
        write!(f, "World")?;  // ✅ 正確處理錯誤
        Ok(())
    }
}
```

### 錯誤 2：使用了錯誤的 Formatter

```rust
// ❌ 錯誤 - 混淆了 write! 宏的多種形式
use std::io::Write;

impl fmt::Display for MyType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // f 是 fmt::Formatter，不是 io::Write
        // 這會編譯錯誤
        f.write_all(b"Hello")?;  // ❌ 不能調用 io::Write 方法
        Ok(())
    }
}

// ✅ 正確 - 使用 write! 宏
impl fmt::Display for MyType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Hello")?;  // ✅ 使用 write! 宏
        Ok(())
    }
}
```

### 錯誤 3：沒有正確處理 Option/Result

```rust
// ❌ 錯誤
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let msg = if let AppError::MdnsError(m) = self { m } else { "unknown" };
        write!(f, "Error: {}", msg)  // ❌ 型別不匹配
    }
}

// ✅ 正確
impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AppError::MdnsError(msg) => write!(f, "mDNS error: {}", msg),
            _ => write!(f, "Unknown error"),
        }
    }
}
```

## 記憶速記

### write!() 在 Display 中的三個重點

```
write!(f, format_string, args...)
   │    │   │               │
   1    2   3               4

1. write! 宏 - Rust 內建的格式化工具
2. f - Formatter 物件（目標輸出）
3. format_string - 格式化字符串
4. args... - 要插入的值（可選）

返回值：fmt::Result
  - Ok(())：成功寫入
  - Err(fmt::Error)：寫入失敗（罕見）
```

### 執行流程速記

```
write!(f, "prefix: {}", value)
    ↓
1. 接收 Formatter f
2. 看見 {} 佔位符
3. 取得 value 的 Display 表現
4. 寫入 f：「prefix: value_string」
5. 返回 Result
```

---

## 相關資源

- [Rust std::fmt::Display 文檔](https://doc.rust-lang.org/std/fmt/trait.Display.html)
- [Rust std::fmt::Formatter 文檔](https://doc.rust-lang.org/std/fmt/struct.Formatter.html)
- [Rust 格式化語法](https://doc.rust-lang.org/std/fmt/index.html)
- [Rust 寫入 I/O](https://doc.rust-lang.org/std/io/trait.Write.html)
