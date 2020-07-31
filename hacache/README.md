## 缓存组件

![image](https://user-images.githubusercontent.com/14919255/86870312-c2304b00-c10a-11ea-873c-252fca21ceaa.png)


### 读取逻辑

1. 如果缓存内容在有效期内，直接返回
2. 如果缓存内容不在有效期内，但是过期的时间在可接受范围内，返回过期的缓存内容，并触发缓存更新任务，后台更新缓存
3. 如果缓存内容不在有效期内，并且不在可接受过期范围内，强制更新缓存，并返回更新后的内容

### 更新逻辑

更新缓存时，需要执行原函数，为了避免缓存穿透带来的雪崩，给执行原函数加了并发限制。

1. 如果上述（2） 步中，异步更新缓存触发原函数调用限流，那么直接跳过；
2. 如果上述（3）步中，强制更新缓存触发限流，那么此时强制返回过期缓存，保证服务可用。