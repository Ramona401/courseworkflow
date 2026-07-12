/**
 * geographyLabTemplates.ts
 *
 * 地理互动实验室模板聚合入口。
 *
 * 当前共30个模板、10个分组：
 *
 * 1. 基础定位与地球运动
 *    - 经纬网、经纬度与半球定位
 *    - 等高线地形图、坡度与地形剖面
 *    - 地球自转、公转与昼夜长短变化
 *
 * 2. 大气运动与天气系统
 *    - 热力环流与海陆风
 *    - 气压带、风带与季节移动
 *    - 冷锋、暖锋和气旋天气过程
 *
 * 3. 水循环、河流与海洋系统
 *    - 水循环过程与人类活动影响
 *    - 河流补给、流量过程线与汛期
 *    - 洋流分布及其影响
 *
 * 4. 地质作用与地貌演化
 *    - 板块边界、地震与火山
 *    - 岩石圈物质循环与三大岩石转化
 *    - 流水侵蚀、搬运、堆积与河流地貌演化
 *
 * 5. 自然环境整体性与地域分异
 *    - 自然环境要素相互作用与整体性
 *    - 纬度地带性与从沿海到内陆的水平地域分异
 *    - 山地垂直地域分异、林线与雪线
 *
 * 6. 人口、聚落与城市发展
 *    - 人口增长、人口结构与人口转变
 *    - 人口迁移、推拉因素与迁移流
 *    - 城市化、城市功能区与城市空间结构
 *
 * 7. 生产活动与地域联系
 *    - 农业区位因素与农业地域类型
 *    - 工业区位因素、工业集聚与产业转移
 *    - 交通运输方式、交通区位与交通网络
 *
 * 8. 区域发展与资源环境
 *    - 流域综合开发、梯级开发与生态治理
 *    - 资源开发、跨区域调配与环境承载力
 *    - 区域发展差异、区域联系与协调发展
 *
 * 9. 自然灾害与地理信息技术
 *    - 台风结构、移动路径与灾害影响
 *    - 洪涝、干旱形成机制与灾害风险
 *    - 遥感、GIS与灾害监测分析
 *
 * 10. 全球变化与可持续发展
 *    - 碳循环、温室效应与全球气候变化
 *    - 海平面上升、沿海风险与适应
 *    - 能源转型、低碳城市与可持续发展
 */

import type {
  GeographyLabTemplate,
} from './geographyLabUtils'

import {
  GEOGRAPHY_LAB_TEMPLATES_LOCATION_EARTH,
} from './geographyLabTemplatesLocationEarth'

import {
  GEOGRAPHY_LAB_TEMPLATES_TOPOGRAPHY,
} from './geographyLabTemplatesTopography'

import {
  GEOGRAPHY_LAB_TEMPLATES_EARTH_MOTION,
} from './geographyLabTemplatesEarthMotion'

import {
  GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_THERMAL,
} from './geographyLabTemplatesAtmosphereThermalCirculation'

import {
  GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_PRESSURE_WIND,
} from './geographyLabTemplatesAtmospherePressureWindBelts'

import {
  GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_FRONT_CYCLONE,
} from './geographyLabTemplatesAtmosphereFrontCyclone'

import {
  GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_WATER_CYCLE,
} from './geographyLabTemplatesHydrologyWaterCycle'

import {
  GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_RIVER_REGIME,
} from './geographyLabTemplatesHydrologyRiverRegime'

import {
  GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_OCEAN_CURRENTS,
} from './geographyLabTemplatesHydrologyOceanCurrents'

import {
  GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_PLATE_BOUNDARY,
} from './geographyLabTemplatesGeologyPlateBoundary'

import {
  GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_ROCK_CYCLE,
} from './geographyLabTemplatesGeologyRockCycle'

import {
  GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_FLUVIAL_LANDFORMS,
} from './geographyLabTemplatesGeologyFluvialLandforms'

import {
  GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_HOLISM,
} from './geographyLabTemplatesEnvironmentHolism'

import {
  GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_HORIZONTAL_ZONATION,
} from './geographyLabTemplatesEnvironmentHorizontalZonation'

import {
  GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_VERTICAL_ZONATION,
} from './geographyLabTemplatesEnvironmentVerticalZonation'

import {
  GEOGRAPHY_LAB_TEMPLATES_HUMAN_POPULATION_TRANSITION,
} from './geographyLabTemplatesHumanPopulationTransition'

import {
  GEOGRAPHY_LAB_TEMPLATES_HUMAN_MIGRATION,
} from './geographyLabTemplatesHumanMigration'

import {
  GEOGRAPHY_LAB_TEMPLATES_HUMAN_URBANIZATION,
} from './geographyLabTemplatesHumanUrbanization'

import {
  GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_AGRICULTURE,
} from './geographyLabTemplatesProductionAgriculture'

import {
  GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_INDUSTRY,
} from './geographyLabTemplatesProductionIndustry'

import {
  GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_TRANSPORT,
} from './geographyLabTemplatesProductionTransport'

import {
  GEOGRAPHY_LAB_TEMPLATES_REGION_WATERSHED,
} from './geographyLabTemplatesRegionWatershed'

import {
  GEOGRAPHY_LAB_TEMPLATES_REGION_RESOURCES,
} from './geographyLabTemplatesRegionResources'

import {
  GEOGRAPHY_LAB_TEMPLATES_REGION_COORDINATION,
} from './geographyLabTemplatesRegionCoordination'

import {
  GEOGRAPHY_LAB_TEMPLATES_DISASTER_TYPHOON,
} from './geographyLabTemplatesDisasterTyphoon'

import {
  GEOGRAPHY_LAB_TEMPLATES_DISASTER_FLOOD_DROUGHT,
} from './geographyLabTemplatesDisasterFloodDrought'

import {
  GEOGRAPHY_LAB_TEMPLATES_DISASTER_REMOTE_SENSING_GIS,
} from './geographyLabTemplatesDisasterRemoteSensingGIS'

import {
  GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_CARBON_CLIMATE,
} from './geographyLabTemplatesSustainabilityCarbonClimate'

import {
  GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_SEA_LEVEL_ADAPTATION,
} from './geographyLabTemplatesSustainabilitySeaLevelAdaptation'

import {
  GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_ENERGY_LOW_CARBON_CITY,
} from './geographyLabTemplatesSustainabilityEnergyLowCarbonCity'

export const GEOGRAPHY_LAB_TEMPLATES:
GeographyLabTemplate[] = [
  ...GEOGRAPHY_LAB_TEMPLATES_LOCATION_EARTH,
  ...GEOGRAPHY_LAB_TEMPLATES_TOPOGRAPHY,
  ...GEOGRAPHY_LAB_TEMPLATES_EARTH_MOTION,
  ...GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_THERMAL,
  ...GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_PRESSURE_WIND,
  ...GEOGRAPHY_LAB_TEMPLATES_ATMOSPHERE_FRONT_CYCLONE,
  ...GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_WATER_CYCLE,
  ...GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_RIVER_REGIME,
  ...GEOGRAPHY_LAB_TEMPLATES_HYDROLOGY_OCEAN_CURRENTS,
  ...GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_PLATE_BOUNDARY,
  ...GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_ROCK_CYCLE,
  ...GEOGRAPHY_LAB_TEMPLATES_GEOLOGY_FLUVIAL_LANDFORMS,
  ...GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_HOLISM,
  ...GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_HORIZONTAL_ZONATION,
  ...GEOGRAPHY_LAB_TEMPLATES_ENVIRONMENT_VERTICAL_ZONATION,
  ...GEOGRAPHY_LAB_TEMPLATES_HUMAN_POPULATION_TRANSITION,
  ...GEOGRAPHY_LAB_TEMPLATES_HUMAN_MIGRATION,
  ...GEOGRAPHY_LAB_TEMPLATES_HUMAN_URBANIZATION,
  ...GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_AGRICULTURE,
  ...GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_INDUSTRY,
  ...GEOGRAPHY_LAB_TEMPLATES_PRODUCTION_TRANSPORT,
  ...GEOGRAPHY_LAB_TEMPLATES_REGION_WATERSHED,
  ...GEOGRAPHY_LAB_TEMPLATES_REGION_RESOURCES,
  ...GEOGRAPHY_LAB_TEMPLATES_REGION_COORDINATION,
  ...GEOGRAPHY_LAB_TEMPLATES_DISASTER_TYPHOON,
  ...GEOGRAPHY_LAB_TEMPLATES_DISASTER_FLOOD_DROUGHT,
  ...GEOGRAPHY_LAB_TEMPLATES_DISASTER_REMOTE_SENSING_GIS,
  ...GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_CARBON_CLIMATE,
  ...GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_SEA_LEVEL_ADAPTATION,
  ...GEOGRAPHY_LAB_TEMPLATES_SUSTAINABILITY_ENERGY_LOW_CARBON_CITY,
]

export function getGeographyLabGroups(): {
  group: string
  items: GeographyLabTemplate[]
}[] {
  const groups: {
    group: string
    items: GeographyLabTemplate[]
  }[] = []

  for (
    const template
    of GEOGRAPHY_LAB_TEMPLATES
  ) {
    let group = groups.find(
      item => item.group === template.group,
    )

    if (!group) {
      group = {
        group: template.group,
        items: [],
      }

      groups.push(group)
    }

    group.items.push(template)
  }

  return groups
}

export function findGeographyLabTemplate(
  id: string,
): GeographyLabTemplate | undefined {
  return GEOGRAPHY_LAB_TEMPLATES.find(
    template => template.id === id,
  )
}
