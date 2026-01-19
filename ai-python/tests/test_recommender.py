import pytest
import asyncio
import sys
sys.path.append('.')

from data import RecommendRequest
from recommender import ProblemRecommender

@pytest.mark.asyncio
async def test_recommend_with_weak_points():
    request = RecommendRequest(
        student_id="2025001",
        weak_points={
            "数组越界": 3,
            "时间复杂度高": 2,
            "边界条件错误": 4,
            "递归深度过大": 1
        },
        max_recommendations=5
    )
    
    recommender = ProblemRecommender()
    result = await recommender.recommend(request)
    
    assert result is not None
    assert result.student_id == "2025001"
    assert len(result.recommendations) > 0
    assert len(result.recommendations) <= 5
    assert hasattr(result, 'analysis')
    
    # 检查每个推荐项
    for rec in result.recommendations:
        assert hasattr(rec, 'tag')
        assert hasattr(rec, 'relevance')
        assert hasattr(rec, 'reason')
        assert 0.0 <= rec.relevance <= 1.0
    
    print(f"测试通过！推荐了{len(result.recommendations)}个标签")
    for rec in result.recommendations:
        print(f"- {rec.tag} (相关度: {rec.relevance:.2f}): {rec.reason}")

@pytest.mark.asyncio
async def test_recommend_empty_weak_points():
    request = RecommendRequest(
        student_id="2025002",
        weak_points={},  # 无薄弱点
        max_recommendations=3
    )
    
    recommender = ProblemRecommender()
    result = await recommender.recommend(request)
    
    assert result is not None
    assert len(result.recommendations) > 0
    print(f"无薄弱点测试通过！推荐: {[r.tag for r in result.recommendations]}")

if __name__ == "__main__":
    asyncio.run(test_recommend_with_weak_points())
    asyncio.run(test_recommend_empty_weak_points())