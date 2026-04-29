**Цель задания**

Реализовать два gRPC-сервиса (OrderService и SpotInstrumentService) для биржевой системы, которые взаимодействуют друг с другом. Сервисы должны использовать предоставленные интерсепторы для трассировки, логирования и обработки ошибок, а также интегрировать Prometheus для мониторинга. Задание направлено на изучение gRPC, работы с интерсепторами, маппингом доменных сущностей и тестированием.

**Описание системы**

- **OrderService**: Управляет заказами (создание заказа, получение статуса).
- **SpotInstrumentService**: Управляет рынками (market), предоставляя информацию о доступных рынках.
- OrderService вызывает SpotInstrumentService для проверки доступности рынка перед созданием заказа.
- Сервисы используют предоставленные интерсепторы для добавления x-request-id, логирования и обработки паник.
- Для мониторинга используется Prometheus.

**Требования**

**1\. Определение gRPC API**

Создайте .proto файл с описанием двух сервисов: OrderService и SpotInstrumentService.

**Требования к** .proto:

- Использовать proto3.
- Определить сообщения и сервисы согласно спецификации ниже.
- Сгенерировать Go-код с помощью protoc.

**Реализация серверов**

Реализуйте два gRPC-сервиса с использованием предоставленных интерсепторов.

**SpotInstrumentService**

- **Метод** ViewMarkets:
    - Принимает user_roles и возвращает список доступных рынков (только те, где enabled: true и deleted_at равно null).
    - Хранить данные в in-memory хранилище (например, map\[string\]Market).
    - Реализовать маппер для преобразования доменных сущностей в proto-структуры.
- **Интерсепторы**:
    - XRequestID: Добавляет x-request-id в контекст.
    - LoggerInterceptor: Логирует запросы с помощью zap.
    - UnaryPanicRecoveryInterceptor: Обрабатывает паники, возвращая ошибку codes.Internal.
    - prometheus.UnaryServerInterceptor: Собирает метрики.

**OrderService**

- **Метод** GetOrderStatus:
    - Принимает order_id и user_id, возвращает статус заказа.
    - Хранить заказы в in-memory хранилище (например, map\[string\]Order).
- **Метод** CreateOrder:
    - Принимает user_id, market_id, order_type, price и quantity.
    - Вызывает SpotInstrumentService.ViewMarkets для проверки, что рынок существует и активен (enabled: true, deleted_at: null).
    - Создает заказ и сохраняет его в in-memory хранилище.
    - Возвращает order_id и статус (например, "created").
- **Интерсепторы**: Те же, что для SpotInstrumentService.

**3\. Реализация маппера**

Создайте маппер для SpotInstrumentService, который преобразует доменные сущности рынка в proto-структуры (Market).

**4\. Настройка gRPC-сервера**

Настройте gRPC-сервер с использованием предоставленных интерсепторов и Prometheus.

**5\. Реализация клиента**

Создайте консольный клиент, который:

- Отправляет запрос CreateOrder в OrderService с указанием user_id, market_id, order_type, price и quantity.
- Использует интерсептор unary.XRequestID для добавления x-request-id.
- Выводит результат (ID заказа и статус).

**6\. Тестирование**

Напишите тесты для проверки:

- Корректности метода ViewMarkets (возвращает только активные рынки).
- Корректности метода CreateOrder (не создает заказ, если рынок не существует или неактивен).
- Наличия x-request-id в логах.
- Работы метрик Prometheus (например, счетчик вызовов ViewMarkets).

**7\. Усложнения**

- Добавить RedisCacheInterceptor для кэширования ответов ViewMarkets (настроить Redis через Docker).
- Реализовать stream-метод StreamOrderUpdates в OrderService, возвращающий обновления статуса заказа в реальном времени.
- Интегрировать Jaeger для трассировки x-request-id между сервисами.
- Добавить проверку user_roles в CreateOrder для ограничения доступа к рынкам.

**8\. Технические требования**

- Использовать Go и библиотеки:
    - google.golang.org/grpc
    - github.com/grpc-ecosystem/go-grpc-prometheus
    - zap
    - github.com/google/uuid
- Хранить данные в in-memory хранилище (map).
- Настроить Prometheus (можно через Docker: prom/prometheus).
- Обработать ошибки:
    - codes.InvalidArgument для неверных входных данных.
    - codes.NotFound для несуществующего рынка.
    - codes.Internal для внутренних ошибок (через UnaryPanicRecoveryInterceptor).
- Логировать запросы с x-request-id.

**10\. Ресурсы**

- Установите protoc: <https://grpc.io/docs/protoc-installation/>
- Документация gRPC для Go: <https://grpc.io/docs/languages/go/>
- Prometheus для Go: <https://prometheus.io/docs/guides/go-application/>
- Пример настройки Prometheus: <https://github.com/prometheus/prometheus>