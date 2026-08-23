import unittest

from app.service import health_payload


class ServiceTest(unittest.TestCase):
    def test_health_payload(self) -> None:
        self.assertEqual(health_payload(), {"status": "ok"})


if __name__ == "__main__":
    unittest.main()
