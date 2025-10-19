import grpc
import matplotlib.pyplot as plt
from datetime import datetime
from matplotlib.animation import FuncAnimation

from temperature_pb2 import StreamRequest
from temperature_pb2_grpc import TemperatureServiceStub

SERVER_ADDRESS = 'localhost:50051'

class TemperatureGraph:
    # Класс для управления данными графика температуры

    def __init__(self, max_points=60):
        self.max_points = max_points
        self.temperatures = []
        self.timestamps = []

    def add_point(self, temp, timestamp):
        # Добавляет новую точку на график
        self.temperatures.append(temp)
        self.timestamps.append(timestamp)

        # Если точек больше максимума - удаляем самую старую
        if len(self.temperatures) > self.max_points:
            self.temperatures.pop(0)
            self.timestamps.pop(0)

class GraphService:
    # Сервис для отображения графика в реальном времени

    def __init__(self):
        self.fig, self.ax = plt.subplots()
        self.data = TemperatureGraph(60)
        self.line, = self.ax.plot([], [], 'b-', linewidth=2)

        self.ax.set_ylabel('Температура (C)', fontsize=12)
        self.ax.set_xlabel('Время', fontsize=12)
        plt.xticks(rotation=45)
        plt.tight_layout()

    def update_graph(self, temperature, timestamp):
        self.data.add_point(temperature, timestamp)

        times = []
        for ts in self.data.timestamps:
            dt = datetime.fromtimestamp(ts.seconds + ts.nanos / 1e9)
            times.append(dt)

        self.line.set_data(times, self.data.temperatures)

        if times and len(self.data.temperatures) > 1:
            temp_range = max(self.data.temperatures) - min(self.data.temperatures)
            margin = max(0.5, temp_range * 0.1)

            self.ax.set_ylim(min(self.data.temperatures) - margin, max(self.data.temperatures) + margin)

        self.ax.relim()
        self.ax.autoscale_view()

        return self.line,


def run():
    print("Подключаемся к серверу...")

    try:
        # В продакшене можно изменить на защищённое соединение
        channel = grpc.insecure_channel(SERVER_ADDRESS)
        stub = TemperatureServiceStub(channel)

        print("Подключено! Запускаем график...")

        graph = GraphService()

        def animate(frame):
            try:
                response = next(stream)
                graph.update_graph(response.value, response.timestamp)
            except StopIteration:
                plt.close()
            return graph.line,
        
        # Создаём поток данных
        stream = stub.StreamTemperature(StreamRequest())

        ani = FuncAnimation(graph.fig, animate, interval=100, blit=False, cache_frame_data=False)

        plt.show()

    except grpc.RpcError as e:
        print(f"Ошибка соединения: {e}")
    except KeyboardInterrupt:
        print("\n Останавливаем сервис...")
    except Exception as e:
        print(f"Неожиданная ошибка: {e}")
    finally:
        print("Завершение работы...")

if __name__ == '__main__':
    run()